package receiver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// Message defines the message structure received from the Signal API.
type Message struct {
	Envelope struct {
		Source         string `json:"source"`
		SourceNumber   string `json:"sourceNumber"`
		SourceUUID     string `json:"sourceUuid"`
		SourceName     string `json:"sourceName"`
		SourceDevice   int    `json:"sourceDevice"`
		Timestamp      int64  `json:"timestamp"`
		ReceiptMessage *struct {
			When       int64   `json:"when"`
			IsDelivery bool    `json:"isDelivery"`
			IsRead     bool    `json:"isRead"`
			IsViewed   bool    `json:"isViewed"`
			Timestamps []int64 `json:"timestamps"`
		} `json:"receiptMessage,omitempty"`
		TypingMessage *struct {
			Action    string `json:"action"`
			Timestamp int64  `json:"timestamp"`
		} `json:"typingMessage,omitempty"`
		DataMessage *struct {
			Timestamp        int64   `json:"timestamp"`
			Message          *string `json:"message"`
			ExpiresInSeconds int     `json:"expiresInSeconds"`
			ViewOnce         bool    `json:"viewOnce"`
			GroupInfo        *struct {
				GroupID   string `json:"groupId"`
				GroupName string `json:"groupName"`
				Revision  int64  `json:"revision"`
				Type      string `json:"type"`
			} `json:"groupInfo,omitempty"`
			Quote *struct {
				ID           int          `json:"id"`
				Author       string       `json:"author"`
				AuthorNumber string       `json:"authorNumber"`
				AuthorUUID   string       `json:"authorUuid"`
				Text         string       `json:"text"`
				Attachments  []Attachment `json:"attachments"`
			} `json:"quote,omitempty"`
			Mentions []struct {
				Name   string `json:"name"`
				Number string `json:"number"`
				UUID   string `json:"uuid"`
				Start  int    `json:"start"`
				Length int    `json:"length"`
			} `json:"mentions,omitempty"`
			Sticker *struct {
				PackID    string `json:"packId"`
				StickerID int    `json:"stickerId"`
			} `json:"sticker,omitempty"`
			Attachments  []Attachment `json:"attachments,omitempty"`
			RemoteDelete *struct {
				Timestamp int64 `json:"timestamp"`
			} `json:"remoteDelete,omitempty"`
		} `json:"dataMessage,omitempty"`
		SyncMessage *struct{} `json:"syncMessage,omitempty"`
	} `json:"envelope"`

	Account string `json:"account"`
}

// Attachment defines the attachment structure of a message.
type Attachment struct {
	ContentType     string  `json:"contentType"`
	ID              string  `json:"id"`
	Filename        *string `json:"filename"`
	Size            int     `json:"size"`
	Width           *int    `json:"width"`
	Height          *int    `json:"height"`
	Caption         *string `json:"caption"`
	UploadTimestamp *int64  `json:"uploadTimestamp"`
}

// Client represents the Signal API client, and is returned by the New() function.
type Client struct {
	uri  *url.URL
	conn *websocket.Conn

	mu       sync.Mutex
	messages []Message
}

// New creates a new Signal API client and returns it.
// An error is returned if a websocket fails to open with the Signal's API
// /v1/receive.
func New(uri *url.URL) (*Client, error) {
	c := &Client{uri: uri}

	return c, c.Connect()
}

func (c *Client) Connect() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	log.Print("Connecting to the Signal API")

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
	log.Print("Starting the receive loop from Signal API")

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("error returned by the websocket: %s", err)

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
		log.Printf("Error decoding the message below: %s", err)
		log.Print(string(msg[:]))

		return
	}

	// Do not record receipt, typing, group update or sync messages, etc.
	if m.Envelope.DataMessage == nil || m.Envelope.DataMessage.Message == nil {
		log.Printf("Ignoring non-data message: %s", string(msg))

		return
	}

	c.mu.Lock()
	c.messages = append(c.messages, m)
	c.mu.Unlock()

	log.Printf("The following message was successfully recorded: %s", string(msg))
}
