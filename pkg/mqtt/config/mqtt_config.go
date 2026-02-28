package config

import (
	"strings"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

const (
	ClientPrefix string = "signal-api-receiver"

	TopicMessageSuffix   string = "message"
	TopicOnlineSuffix    string = "online"
	TopicConnectedSuffix string = "connected"

	sessionExpiryInterval                 uint32 = 60
	keepAlive                             uint16 = 20
	statusRetain                          bool   = true
	statusQosValue                        byte   = 0
	connectionTimeout                            = 10 * time.Second
	connectionTimeoutInitial                     = 5 * time.Second
	reconnectDelay                               = 7 * time.Second
	cleanStartOnInitialConnectionFallback bool   = false
	lastWillDelayInterval                 uint32 = 80
	statusOnlinePayload                   string = "online"
	statusOfflinePayload                  string = "offline"
	payloadContentType                    string = "application/json"
)

func QosValues() []int {
	return []int{0, 1, 2}
}

type InitOptions struct {
	Server              string
	ClientID            string
	User                string
	Password            string
	TopicPrefix         string
	Qos                 int
	RetainMessages      bool
	ValidateCertificate bool
}

type Topics struct {
	Message string
	Status  string
}

func New(options InitOptions) *Config {
	var (
		payloadFormat          byte = 1
		rLastWillDelayInterval      = lastWillDelayInterval
	)

	return &Config{
		InitOptions: options,

		ClientPrefix:             ClientPrefix,
		SessionExpiryInterval:    sessionExpiryInterval,
		KeepAlive:                keepAlive,
		StatusRetain:             statusRetain,
		StatusQosValue:           statusQosValue,
		ConnectionTimeout:        connectionTimeout,
		ConnectionTimeoutInitial: connectionTimeoutInitial,
		ReconnectDelay:           reconnectDelay,

		Topics:                        marshalTopics(options.TopicPrefix),
		CleanStartOnInitialConnection: cleanStartOnInitialConnection(options.Qos),

		StatusOnlinePayload:  []byte(statusOnlinePayload),
		StatusOfflinePayload: []byte(statusOfflinePayload),

		PublishProperties: &paho.PublishProperties{
			PayloadFormat: &payloadFormat,
			ContentType:   payloadContentType,
		},

		WillProperties: &paho.WillProperties{
			PayloadFormat:     &payloadFormat,
			ContentType:       payloadContentType,
			WillDelayInterval: &rLastWillDelayInterval,
		},
	}
}

type Config struct {
	InitOptions
	ClientPrefix                  string
	SessionExpiryInterval         uint32
	KeepAlive                     uint16
	StatusRetain                  bool
	StatusQosValue                byte
	ConnectionTimeout             time.Duration
	ConnectionTimeoutInitial      time.Duration
	ReconnectDelay                time.Duration
	CleanStartOnInitialConnection bool
	StatusOnlinePayload           []byte
	StatusOfflinePayload          []byte
	Topics                        *Topics
	PublishProperties             *paho.PublishProperties
	WillProperties                *paho.WillProperties
}

func (c Config) QosValue() byte {
	return byte(c.Qos)
}

func (c Config) GetStatusPayloadForState(state bool) []byte {
	payload := c.StatusOnlinePayload

	if !state {
		payload = c.StatusOfflinePayload
	}

	return payload
}

//nolint:unparam
func cleanStartOnInitialConnection(qos int) bool {
	if qos != 0 {
		return false
	}

	return cleanStartOnInitialConnectionFallback
}

func marshalTopics(topicPrefix string) *Topics {
	topicPrefix = strings.Trim(topicPrefix, "#/ ")

	if topicPrefix == "" {
		topicPrefix = ClientPrefix
	}

	return &Topics{
		Message:   topicPrefix + "/" + TopicMessageSuffix,
		Status:    topicPrefix + "/" + TopicOnlineSuffix,
		Connected: topicPrefix + "/" + TopicConnectedSuffix,
	}
}
