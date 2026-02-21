package mqtt

import (
	"context"
	"crypto/tls"
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

const (
	topicMessageSuffix                   = "message"
	topicStatusSuffix                    = "online"
	cleanStartOnInitialConnection bool   = false
	sessionExpiryInterval         uint32 = 60
	keepAlive                     uint16 = 20
	payloadContentType            string = "application/json"
	statusOnlinePayload           string = "true"
	statusOfflinePayload          string = "false"
	statusRetain                  bool   = true
	statusQosValue                byte   = 0
)

var (
	// ErrMqttConnectionAttempt is thrown when MQTT connection attempt has failed.
	ErrMqttConnectionAttempt = errors.New("mqtt connection attempt error")

	// ErrMqttConnectionFailed is thrown when waiting for connection has failed.
	ErrMqttConnectionFailed = errors.New("mqtt connection error")

	//nolint:gochecknoglobals
	connectionTimeout = 10 * time.Second

	//nolint:gochecknoglobals
	initialConnectionTimeout = 5 * time.Second

	//nolint:gochecknoglobals
	reconnectDelay = 7 * time.Second

	//nolint:gochecknoglobals
	lastWillDelayInterval = uint32(sessionExpiryInterval + uint32(keepAlive))

	//nolint:gochecknoglobals
	statusPayloadFormat byte = 1
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
	Server              string
	ClientID            string
	User                string
	Password            string
	TopicPrefix         string
	Qos                 int
	ValidateCertificate bool
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

	var conn *autopaho.ConnectionManager

	conn, err = autopaho.NewConnection(ctx, autopaho.ClientConfig{
		ServerUrls: []*url.URL{serverURL},
		TlsCfg: &tls.Config{
			InsecureSkipVerify: !config.ValidateCertificate,
		},
		ConnectUsername:               config.User,
		ConnectPassword:               []byte(config.Password),
		CleanStartOnInitialConnection: cleanStartOnInitialConnection,
		SessionExpiryInterval:         sessionExpiryInterval,
		KeepAlive:                     keepAlive,
		ConnectTimeout:                connectionTimeout,
		ReconnectBackoff: func(attempt int) time.Duration {
			switch attempt {
			case 0:
				return 0
			default:
				return reconnectDelay
			}
		},
		OnConnectionUp: func(manager *autopaho.ConnectionManager, _ *paho.Connack) {
			logger.Info().
				Str("clientID", config.ClientID).
				Msg("MQTT: Connection successfully established.")

			publishOnlineState(ctx, manager, &config, statusOnlinePayload)
		},
		OnConnectionDown: func() bool {
			publishOnlineState(ctx, conn, &config, statusOfflinePayload)

			return true
		},
		OnConnectError: func(err error) {
			logger.Error().Err(err).
				Str("reconnect_in", strconv.FormatFloat(reconnectDelay.Seconds(), 'f', 0, 64)+"sec").
				Msg("MQTT: Error whilst attempting MQTT connection")
		},
		WillMessage: &paho.WillMessage{
			Retain:  statusRetain,
			QoS:     statusQosValue,
			Topic:   strings.Join([]string{config.TopicPrefix, topicStatusSuffix}, "/"),
			Payload: []byte(statusOfflinePayload),
		},
		WillProperties: &paho.WillProperties{
			PayloadFormat:     &statusPayloadFormat,
			ContentType:       payloadContentType,
			WillDelayInterval: &lastWillDelayInterval,
		},
		ClientConfig: paho.ClientConfig{
			ClientID: config.ClientID,
			OnClientError: func(err error) {
				logger.Error().Err(err).Msg("MQTT: Client error")
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if isUnrecoverableReasonCodeError(d.ReasonCode) {
					logger.Error().Msgf("Cancel reconnect. Server disconnected with unrecoverable reason-code %d.", d.ReasonCode)
					_ = conn.Disconnect(ctx)
				} else {
					if d.Properties != nil {
						logger.Error().Msgf("MQTT: Server requested disconnect: %s", d.Properties.ReasonString)
					} else {
						logger.Error().Msgf("MQTT: Server requested disconnect; reason code: %d", d.ReasonCode)
					}
				}
			},
			PublishHook: func(publish *paho.Publish) {
				logger.Debug().
					Bool("retain", publish.Retain).
					Bytes("payload", publish.Payload).
					Msg("MQTT: A message was published to " + publish.Topic)
			},
		},
	})
	// Initial connect will return unrecoverable Connack error
	if err != nil {
		return fmt.Errorf(
			"%w: error whilst attempting mqtt connection: %w",
			ErrMqttConnectionAttempt,
			err,
		)
	}
	// Register with notifier
	notifier.RegisterHandler(ctx, handlerOpt{
		Logger: logger,
		Topic:  strings.Join([]string{config.TopicPrefix, topicMessageSuffix}, "/"),
		Config: handlerConfig{
			Qos:         config.Qos,
			TopicPrefix: config.TopicPrefix,
		},
		Manager: conn,
	})

	waitCtx, waitCancel := context.WithTimeout(ctx, initialConnectionTimeout)
	defer waitCancel()

	if err = conn.AwaitConnection(waitCtx); err != nil && errors.Is(err, context.DeadlineExceeded) == false {
		// The initial connection may be slow, but anything that cancels its context is unrecoverable for us too.
		return fmt.Errorf(
			"%w: mqtt error while waiting for connection: %w",
			ErrMqttConnectionFailed,
			err,
		)
	}

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
		m.Logger.Error().Err(err).Msg("MQTT: Error while marshaling message")

		return err
	}

	publishOptions := &paho.Publish{
		QoS:    byte(m.Config.Qos),
		Topic:  m.Topic,
		Retain: false,
		Properties: &paho.PublishProperties{
			PayloadFormat: &payloadFormat,
			ContentType:   payloadContentType,
		},
		Payload: payload,
	}

	_, err = m.Manager.Publish(ctx, publishOptions)

	if errors.Is(err, autopaho.ConnectionDownError) {
		m.Logger.Warn().
			AnErr("publish_error", autopaho.ConnectionDownError).
			Msg("MQTT: Connection issues while publishing - using queue to postpone.")

		err = m.Manager.PublishViaQueue(ctx, &autopaho.QueuePublish{Publish: publishOptions})
	}

	if err != nil {
		m.Logger.Error().Err(err).Msg("MQTT: Error while publishing message")

		return err
	}

	return nil
}

func publishOnlineState(ctx context.Context, manager *autopaho.ConnectionManager, config *InitConfig, payload string) {
	go func() {
		_, err := manager.Publish(ctx, &paho.Publish{
			QoS:    statusQosValue,
			Topic:  strings.Join([]string{config.TopicPrefix, topicStatusSuffix}, "/"),
			Retain: statusRetain,
			Properties: &paho.PublishProperties{
				PayloadFormat: &statusPayloadFormat,
				ContentType:   payloadContentType,
			},
			Payload: []byte(payload),
		})
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("MQTT: Error while publishing online state")
		}
	}()
}
