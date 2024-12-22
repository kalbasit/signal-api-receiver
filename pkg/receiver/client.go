package receiver

import (
	"context"
	"encoding/json"
	"fmt"
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

	mu       sync.Mutex
	messages []Message
}

// New creates a new Signal API client and returns it.
// An error is returned if a websocket fails to open with the Signal's API
// /v1/receive.
func New(ctx context.Context, uri *url.URL) (*Client, error) {
	c := &Client{uri: uri, logger: *zerolog.Ctx(ctx)}

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
func (c *Client) ReceiveLoop() error {
	log := c.logger.With().Str("func", "ReceiveLoop").Logger()

	log.Info().Msg("Starting the receive loop from Signal API")

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("error returned by the websocket")

			return err
		}

		c.recordMessage(msg)
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

func (c *Client) recordMessage(msg []byte) {
	var m Message
	if err := json.Unmarshal(msg, &m); err != nil {
		c.logger.
			Error().
			Err(err).
			Str("signal-message", string(msg)).
			Msg("error decoding the message")

		return
	}

	// Do not record receipt, typing, group update or sync messages, etc.
	if m.Envelope.DataMessage == nil || m.Envelope.DataMessage.Message == nil {
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
