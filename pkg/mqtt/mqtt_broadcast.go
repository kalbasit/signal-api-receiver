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
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/rs/zerolog"

	"github.com/kalbasit/signal-api-receiver/pkg/mqtt/config"
	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
)

var (
	// ErrMqttConnectionAttempt is thrown when MQTT connection attempt has failed.
	ErrMqttConnectionAttempt = errors.New("mqtt connection attempt error")

	// ErrMqttConnectionFailed is thrown when waiting for connection has failed.
	ErrMqttConnectionFailed = errors.New("mqtt connection error")
)

type publishPayload struct {
	Message *receiver.Message `json:"content"`
	Types   []string          `json:"types"`
}

type handlerOpt struct {
	Logger      zerolog.Logger
	Config      *config.Config
	Manager     *autopaho.ConnectionManager
	connState   int32
	connStateMu sync.Mutex
}

const (
	connStateUnknown int32 = -1
	connStateOffline int32 = 0
	connStateOnline  int32 = 1
)

func Init(
	ctx context.Context,
	notifier *receiver.Notifier,
	options config.InitOptions,
) error {
	logger := *zerolog.Ctx(ctx)
	logger = logger.With().Str("scope", "MQTT").Logger()

	if !strings.Contains(options.Server, "://") {
		options.Server = "mqtt://" + options.Server
	}

	serverURL, err := url.Parse(options.Server)
	if err != nil {
		logger.Error().Err(err).Msgf("Error while parsing the server url %s", options.Server)

		return err
	}

	cfg := config.New(options)

	var conn *autopaho.ConnectionManager

	conn, err = autopaho.NewConnection(ctx, autopaho.ClientConfig{
		ServerUrls: []*url.URL{serverURL},
		TlsCfg: &tls.Config{
			InsecureSkipVerify: options.InsecureSkipVerify, //nolint:gosec
		},
		ConnectUsername:               options.User,
		ConnectPassword:               []byte(options.Password),
		CleanStartOnInitialConnection: cfg.CleanStartOnInitialConnection,
		SessionExpiryInterval:         cfg.SessionExpiryInterval,
		KeepAlive:                     cfg.KeepAlive,
		ConnectTimeout:                cfg.ConnectionTimeout,
		ReconnectBackoff: func(attempt int) time.Duration {
			switch attempt {
			case 0:
				return 0
			default:
				return cfg.ReconnectDelay
			}
		},
		OnConnectionUp: func(manager *autopaho.ConnectionManager, _ *paho.Connack) {
			logger.Info().
				Str("clientID", options.ClientID).
				Msg("Connection successfully established.")

			publishOnlineState(ctx, manager, cfg, true)
		},
		OnConnectionDown: func() bool {
			logger.Info().
				Str("clientID", options.ClientID).
				Msg("Connection has been lost.")

			return true
		},
		OnConnectError: func(err error) {
			logger.Error().Err(err).
				Str("reconnect_in", strconv.FormatFloat(cfg.ReconnectDelay.Seconds(), 'f', 0, 64)+"sec").
				Msg("Error whilst attempting MQTT connection")
		},
		WillMessage: &paho.WillMessage{
			Retain:  cfg.StatusRetain,
			QoS:     cfg.StatusQosValue,
			Topic:   cfg.Topics.Status,
			Payload: cfg.StatusOfflinePayload,
		},
		WillProperties: cfg.WillProperties,
		ClientConfig: paho.ClientConfig{
			ClientID: options.ClientID,
			OnClientError: func(err error) {
				logger.Error().Err(err).Msg("Client error")
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if isUnrecoverableReasonCodeError(d.ReasonCode) {
					logger.Error().Msgf("Cancel reconnect. Server disconnected with unrecoverable reason-code %d.", d.ReasonCode)

					_ = conn.Disconnect(ctx)
				} else {
					if d.Properties != nil {
						logger.Error().Msgf("Server requested disconnect: %s", d.Properties.ReasonString)
					} else {
						logger.Error().Msgf("Server requested disconnect; reason code: %d", d.ReasonCode)
					}
				}
			},
			PublishHook: func(publish *paho.Publish) {
				logger.Debug().
					Bool("retain", publish.Retain).
					Bytes("payload", publish.Payload).
					Msg("A message was published to " + publish.Topic)
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

	registerNotifier(ctx, notifier, &handlerOpt{
		Logger:  logger,
		Config:  cfg,
		Manager: conn,
	})

	waitCtx, waitCancel := context.WithTimeout(ctx, cfg.ConnectionTimeoutInitial)
	defer waitCancel()

	if err = conn.AwaitConnection(waitCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		// The initial connection may be slow, but anything that cancels its context is unrecoverable for us too.
		return fmt.Errorf(
			"%w: mqtt error while waiting for connection: %w",
			ErrMqttConnectionFailed,
			err,
		)
	}

	return nil
}

func registerNotifier(ctx context.Context, notifier *receiver.Notifier, options *handlerOpt) {
	options.connState = connStateUnknown
	notifier.RegisterHandler(ctx, options)
}

func (m *handlerOpt) Handle(ctx context.Context, messagePayload receiver.NotifierPayload) error {
	var err error

	if messagePayload.Message != nil {
		err = m.publishMessage(ctx, messagePayload)
	}

	desiredConnState := connStateOffline
	if *messagePayload.IsConnected {
		desiredConnState = connStateOnline
	}

	m.connStateMu.Lock()
	defer m.connStateMu.Unlock()

	if m.connState == connStateUnknown || m.connState != desiredConnState {
		sErr := m.publishConnectionState(ctx, messagePayload)
		if sErr == nil {
			m.connState = desiredConnState
		} else {
			err = errors.Join(sErr, err)
		}
	}

	return err
}

func (m *handlerOpt) publishMessage(ctx context.Context, mPayload receiver.NotifierPayload) error {
	m.Logger.Debug().
		Str("account", mPayload.Message.Account).
		Str("source", mPayload.Message.Envelope.Source).
		Strs("messageTypes", mPayload.Message.MessageTypesStrings()).
		Msg("Broadcast new message")

	payload, err := json.Marshal(
		publishPayload{
			Message: mPayload.Message,
			Types:   mPayload.Message.MessageTypesStrings(),
		},
	)
	if err != nil {
		m.Logger.Error().Err(err).Msg("Error while marshaling message")

		return err
	}

	return publish(ctx, m.Manager, &paho.Publish{
		QoS:        m.Config.QosValue(),
		Topic:      m.Config.Topics.Message,
		Retain:     m.Config.RetainMessages,
		Properties: m.Config.PublishProperties,
		Payload:    payload,
	}, true)
}

func (m *handlerOpt) publishConnectionState(ctx context.Context, payload receiver.NotifierPayload) error {
	return publish(ctx, m.Manager, &paho.Publish{
		QoS:        m.Config.StatusQosValue,
		Topic:      m.Config.Topics.Connected,
		Retain:     m.Config.StatusRetain,
		Properties: m.Config.PublishProperties,
		Payload:    m.Config.GetStatusPayloadForState(*payload.IsConnected),
	}, true)
}

func publishOnlineState(ctx context.Context, manager *autopaho.ConnectionManager, cfg *config.Config, state bool) {
	_ = publish(ctx, manager, &paho.Publish{
		QoS:        cfg.StatusQosValue,
		Topic:      cfg.Topics.Status,
		Retain:     cfg.StatusRetain,
		Properties: cfg.PublishProperties,
		Payload:    cfg.GetStatusPayloadForState(state),
	}, false)
}

func publish(
	ctx context.Context,
	manager *autopaho.ConnectionManager,
	publishOptions *paho.Publish,
	enqueue bool,
) error {
	_, err := manager.Publish(ctx, publishOptions)

	if enqueue && errors.Is(err, autopaho.ConnectionDownError) {
		zerolog.Ctx(ctx).Debug().
			AnErr("m", autopaho.ConnectionDownError).
			Interface("id", publishOptions.PacketID).
			Msg("Message enqueued")

		err = manager.PublishViaQueue(ctx, &autopaho.QueuePublish{Publish: publishOptions})
		manager.Done()
	}

	if err != nil {
		zerolog.Ctx(ctx).Error().
			Err(err).Msg("Error while publishing")
	}

	return err
}
