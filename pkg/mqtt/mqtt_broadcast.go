package mqtt

import (
	"context"
	"net/url"
	"strings"

	"encoding/json"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/leptoquark1/signal-api-receiver/pkg/errors"
	"github.com/leptoquark1/signal-api-receiver/pkg/receiver"
	"github.com/rs/zerolog"
)

type config struct {
	Qos         int
	TopicPrefix string
}

type newMessageNotifier struct {
	Logger  zerolog.Logger
	Config  config
	Topic   string
	Manager *autopaho.ConnectionManager
	ctx     context.Context
}

func Init(
	ctx context.Context,
	server string,
	clientID string,
	user string,
	password string,
	topicPrefix string,
	qos int,
) error {
	logger := *zerolog.Ctx(ctx)

	if !strings.HasPrefix(server, "mqtt://") {
		server = strings.Join([]string{"mqtt://", server}, "")
	}
	serverURL, err := url.Parse(server)

	if err != nil {
		logger.Error().Msgf("error while parsing the MQTT server url %s: %v", server, err)

		return err
	}

	conn, err := autopaho.NewConnection(ctx, autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverURL},
		ConnectUsername:               user,
		ConnectPassword:               []byte(password),
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		KeepAlive:                     20,
		OnConnectionUp: func(_ *autopaho.ConnectionManager, connAck *paho.Connack) {
			logger.Info().Msg("MQTT: connection successfully established.")
		},
		OnConnectError: func(err error) {
			logger.Error().Msgf("error whilst attempting MQTT connection: %s\n", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			OnClientError: func(err error) {
				logger.Error().Msgf("MQTT client error: %s\n", err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					logger.Error().Msgf("MQTT server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					logger.Error().Msgf("MQTT server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		},
	})

	if err != nil {
		return errors.MqttConnectionAttemptError(err)
	}

	if err = conn.AwaitConnection(ctx); err != nil {
		return errors.MqttConnectionFailedError(err)
	}

	receiver.NewMessage.Register(newMessageNotifier{
		Logger: logger,
		Topic:  strings.Join([]string{strings.Trim(topicPrefix, "#/ "), "message"}, "/"),
		Config: config{
			Qos:         qos,
			TopicPrefix: topicPrefix,
		},
		Manager: conn,
		ctx:     ctx,
	})

	return nil
}

type publishPayload struct {
	Message *receiver.Message `json:"content"`
	Types   []string          `json:"types"`
}

func (m newMessageNotifier) Handle(messagePayload receiver.NewMessagePayload) {
	m.Logger.Debug().Msg("MQTT: Broadcast new message")

	payloadFormat := byte(1)
	payload, err := json.Marshal(
		publishPayload{Message: &messagePayload.Message, Types: messagePayload.Message.MessageTypesStrings()},
	)

	if err != nil {
		m.Logger.Error().Err(err).Msg("error while stringify message")
		return
	}

	_, err = m.Manager.Publish(m.ctx, &paho.Publish{
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
		m.Logger.Error().Err(err).Msgf("error while publishing message")

		return
	}
}
