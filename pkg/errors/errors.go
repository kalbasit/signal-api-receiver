package errors

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidSignalAccount        = errors.New("invalid signal account")
	ErrSchemeMissing               = errors.New("scheme is missing")
	ErrMqttQosValueNotAllowed      = errors.New("mqtt qos value is not allowed")
	ErrMqttConnectionFailed        = errors.New("mqtt connection error")
	ErrMqttConnectionAttempt       = errors.New("mqtt connection attempt error")
	ErrMqttInitError               = errors.New("mqtt initialization error")
	ErrLogLevelFormatError         = errors.New("log level format error")
	ErrMessageParseError           = errors.New("message parse error")
	ErrWebSocketConnection         = errors.New("websocket connection error")
	ErrHttpListenerStartError      = errors.New("http listener start error")
	ErrMessageTypeParseFormatError = errors.New("message type parse format error")
	ErrSignalURLParseError         = errors.New("signal url parse error")
	ErrReceiverCreateError         = errors.New("receiver create error")
)

func InvalidSignalAccountError() error {
	return fmt.Errorf(
		"%w: phone number must have leading + followed only by numbers",
		ErrInvalidSignalAccount,
	)
}

func MqttQosValueNotAllowedError(q int, allowedValues []int) error {
	return fmt.Errorf(
		"%w: %d, allowed values are %v",
		ErrMqttQosValueNotAllowed,
		q,
		allowedValues,
	)
}

func MqttConnectionAttemptError(err error) error {
	return fmt.Errorf(
		"%w: error whilst attempting mqtt connection: %w",
		ErrMqttConnectionAttempt,
		err,
	)
}

func MqttConnectionFailedError(err error) error {
	return fmt.Errorf(
		"%w: mqtt error while waiting for connection: %w",
		ErrMqttConnectionFailed,
		err,
	)
}

func MqttInitError(err error) error {
	return fmt.Errorf(
		"%w: error initializing mqtt: %w",
		ErrMqttInitError,
		err,
	)
}

func LogLevelFormatError(lvl string) error {
	return fmt.Errorf(
		"%w: error parsing the log-level %q",
		ErrLogLevelFormatError,
		lvl,
	)
}

func MessageParseError(mts string, err error) error {
	return fmt.Errorf(
		"%w: could not parse message type %q: %w",
		ErrMessageParseError,
		mts,
		err,
	)
}

func WebSocketConnectionError(err error) error {
	return fmt.Errorf(
		"%w: error creating a new websocket connection: %w",
		ErrWebSocketConnection,
		err,
	)
}

func HttpListenerStartError(err error) error {
	return fmt.Errorf(
		"%w: error starting the HTTP listener: %w",
		ErrHttpListenerStartError,
		err,
	)
}

func MessageTypeParseFormatError(mt string, err error) error {
	return fmt.Errorf(
		"%w: could not parse message type %q: %w",
		ErrMessageTypeParseFormatError,
		mt,
		err,
	)
}

func SignalURLParseError(signalAPIURL string, err error) error {
	return fmt.Errorf(
		"%w: error parsing the url %q: %w",
		ErrSignalURLParseError,
		signalAPIURL,
		err,
	)
}

func ReceiverCreateError(err error) error {
	return fmt.Errorf(
		"%w: error creating a new receiver: %w",
		ErrReceiverCreateError,
		err,
	)
}
