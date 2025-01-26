package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/rs/zerolog"

	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
)

var (
	// ErrMqttConnectionAttempt is thrown when MQTT connection attempt has failed.
	ErrMqttConnectionAttempt = errors.New("mqtt connection attempt error")

	// ErrMqttConnectionFailed is thrown when waiting for connection has failed.
	ErrMqttConnectionFailed = errors.New("mqtt connection error")

	initialConnectionTimeout = 5 * time.Second
	reconnectDelay           = 10 * time.Second
)

type handlerConfig struct {
	Qos         int
	TopicPrefix string
}

type handlerOpt struct {
	Logger  zerolog.Logger
	Config  handlerConfig
	Topic   string
	Manager *autopaho.ConnectionManager
}

type InitConfig struct {
	Server      string
	ClientID    string
	User        string
	Password    string
	TopicPrefix string
	Qos         int
}

func Init(
	ctx context.Context,
	notifier *receiver.MessageNotifier,
	config InitConfig,
) error {
	logger := *zerolog.Ctx(ctx)

	if !strings.Contains(config.Server, "://") {
		config.Server = "mqtt://" + config.Server
	}

	serverURL, err := url.Parse(config.Server)
	if err != nil {
		logger.Error().Err(err).Msgf("MQTT: Error while parsing the server url %s", config.Server)

		return err
	}

	topic := strings.Join([]string{strings.Trim(config.TopicPrefix, "#/ "), "message"}, "/")

	conn, err := autopaho.NewConnection(ctx, autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverURL},
		ConnectUsername:               config.User,
		ConnectPassword:               []byte(config.Password),
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		KeepAlive:                     20,
		ReconnectBackoff: func(i int) time.Duration {
			return reconnectDelay
		},
		OnConnectionUp: func(_ *autopaho.ConnectionManager, _ *paho.Connack) {
			logger.Info().
				Str("clientID", config.ClientID).
				Str("topic", topic).
				Str("qos", strconv.Itoa(config.Qos)).
				Msg("MQTT: Connection successfully established.")
		},
		OnConnectError: func(err error) {
			logger.Error().Err(err).
				Dur("reconnect", reconnectDelay).
				Msg("MQTT: Error whilst attempting MQTT connection")
		},
		ClientConfig: paho.ClientConfig{
			ClientID: config.ClientID,
			OnClientError: func(err error) {
				logger.Error().Err(err).Msg("MQTT: Client error")
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					logger.Error().Msgf("MQTT: Server requested disconnect: %s", d.Properties.ReasonString)
				} else {
					logger.Error().Msgf("MQTT: Server requested disconnect; reason code: %d", d.ReasonCode)
				}
			},
		},
	})
	if err != nil {
		return fmt.Errorf(
			"%w: error whilst attempting mqtt connection: %w",
			ErrMqttConnectionAttempt,
			err,
		)
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, initialConnectionTimeout)
	defer waitCancel()

	if err = conn.AwaitConnection(waitCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn().
				Err(err).
				Dur("timeout", initialConnectionTimeout).
				Dur("reconnect", reconnectDelay).
				Msg("MQTT: Initial connection timed out; continuing startup")
		} else {
			return fmt.Errorf(
				"%w: mqtt error while waiting for connection: %w",
				ErrMqttConnectionFailed,
				err,
			)
		}
	}

	notifier.RegisterHandler(ctx, handlerOpt{
		Logger: logger,
		Topic:  topic,
		Config: handlerConfig{
			Qos:         config.Qos,
			TopicPrefix: config.TopicPrefix,
		},
		Manager: conn,
	})

	return nil
}

type publishPayload struct {
	Message *receiver.Message `json:"content"`
	Types   []string          `json:"types"`
}

func (m handlerOpt) Handle(ctx context.Context, messagePayload receiver.MessageNotifierPayload) error {
	m.Logger.Debug().
		Str("account", messagePayload.Message.Account).
		Strs("messageTypes", messagePayload.Message.MessageTypesStrings()).
		Msg("MQTT: Broadcast new message")

	payloadFormat := byte(1)

	payload, err := json.Marshal(
		publishPayload{Message: &messagePayload.Message, Types: messagePayload.Message.MessageTypesStrings()},
	)
	if err != nil {
		m.Logger.Error().Err(err).Msg("MQTT: Error while stringify message")

		return err
	}

	_, err = m.Manager.Publish(ctx, &paho.Publish{
		QoS:    byte(m.Config.Qos),
		Topic:  m.Topic,
		Retain: false,
		Properties: &paho.PublishProperties{
			PayloadFormat: &payloadFormat,
			ContentType:   "application/json",
		},
		Payload: payload,
	})
	if err != nil {
		m.Logger.Error().Err(err).Msg("MQTT: Error while publishing message")

		return err
	}

	return nil
}
