package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// Client represents the Signal API client, and is returned by the New() function.
type Client struct {
	uri  *url.URL
	conn *websocket.Conn

	logger zerolog.Logger

	recordedMessageTypesStrs []string
	recordedMessageTypes     map[MessageType]bool

	mu       sync.Mutex
	messages []Message

	MessageNotifier *MessageNotifier
	notifierTrigger MessageNotifierTrigger
}

// New creates a new Signal API client and returns it.
// An error is returned if a websocket fails to open with the Signal's API
// /v1/receive.
func New(ctx context.Context, uri *url.URL, messageTypes ...string) (*Client, error) {
	notifier, notifierTrigger := InitNotifier(ctx)

	c := &Client{
		uri:                      uri,
		logger:                   *zerolog.Ctx(ctx),
		recordedMessageTypesStrs: messageTypes,
		recordedMessageTypes:     make(map[MessageType]bool),
		MessageNotifier:          notifier,
		notifierTrigger:          notifierTrigger,
	}

	for _, mts := range messageTypes {
		mt, err := ParseMessageType(mts)
		if err != nil {
			return nil, fmt.Errorf("could not parse message type %q: %w", mts, err)
		}

		c.recordedMessageTypes[mt] = true
	}

	return c, c.Connect()
}

func (c *Client) Connect() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.logger.Info().Msg("Connecting to the Signal API")

	conn, _, err := websocket.DefaultDialer.Dial(c.uri.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("error creating a new websocket connetion: %w", err)
	}

	c.conn = conn

	return nil
}

// ReceiveLoop is a blocking call and it loop over receiving messages over the
// websocket and record them internally to be consumed by either Pop() or
// Flush().
func (c *Client) ReceiveLoop(ctx context.Context) error {
	log := c.logger.With().Str("func", "ReceiveLoop").Logger()

	log.
		Info().
		Strs("recorded-message-types", c.recordedMessageTypesStrs).
		Msg("Starting the receive loop from Signal API")

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("error returned by the websocket")

			return err
		}

		c.recordMessage(ctx, msg)
	}
}

// Flush empties out the internal queue of messages and returns them.
func (c *Client) Flush() []Message {
	c.mu.Lock()
	msgs := c.messages
	c.messages = nil
	c.mu.Unlock()

	return msgs
}

// Pop returns the oldest message in the queue or null if no message was found.
func (c *Client) Pop() *Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.messages) == 0 {
		return nil
	}

	msg := c.messages[0]
	c.messages = c.messages[1:]

	return &msg
}

// LocalAddr returns connection local address.
func (c *Client) LocalAddr() *net.TCPAddr {
	return c.conn.LocalAddr().(*net.TCPAddr)
}

func (c *Client) recordMessage(ctx context.Context, msg []byte) {
	var m Message
	if err := json.Unmarshal(msg, &m); err != nil {
		c.logger.
			Error().
			Err(err).
			Str("signal-message", string(msg)).
			Msg("error decoding the message")

		return
	}

	if !c.shouldRecordMessage(m) {
		//nolint:zerologlint
		if c.logger.Debug().Enabled() {
			c.logger.
				Debug().
				Strs("message-types", m.MessageTypesStrings()).
				Interface("message-content", m).
				Msg("ignoring non-data message")
		} else {
			c.logger.
				Info().
				Strs("message-types", m.MessageTypesStrings()).
				Msg("ignoring non-data message")
		}

		return
	}

	c.mu.Lock()
	c.messages = append(c.messages, m)
	c.mu.Unlock()

	err := c.notifierTrigger(ctx, MessageNotifierPayload{Message: m})
	if err != nil {
		c.logger.Error().Err(err).Msg("error while handling new-message")
	}

	//nolint:zerologlint
	if c.logger.Debug().Enabled() {
		c.logger.
			Debug().
			Strs("message-types", m.MessageTypesStrings()).
			Interface("message-content", m).
			Msg("a signal message was successfully recorded")
	} else {
		c.logger.
			Info().
			Strs("message-types", m.MessageTypesStrings()).
			Msg("a signal message was successfully recorded")
	}
}

func (c *Client) shouldRecordMessage(m Message) bool {
	for _, mt := range m.MessageTypes() {
		if c.recordedMessageTypes[mt] {
			return true
		}
	}

	return false
}
