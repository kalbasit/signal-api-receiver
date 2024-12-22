package receiver

import (
	"errors"
	"fmt"
)

// ErrMessageTypeUnknown is returned if message type (string) is not known.
var ErrMessageTypeUnknown = errors.New("message type is unknown")

// MessageType represents a type of a message.
type MessageType uint8

const (
	MessageTypeUnknown MessageType = iota

	// MessageTypeReceipt represents a message that has a receipt.
	MessageTypeReceipt

	// MessageTypeTyping represents a message that has a typing.
	MessageTypeTyping

	// MessageTypeData represents a message that has data.
	MessageTypeData

	// MessageTypeDataMessage represents a message that has data and has a message.
	MessageTypeDataMessage

	// MessageTypeSync represents a message that has a sync.
	MessageTypeSync
)

// AllMessageTypes returns all valid message types.
func AllMessageTypes() []MessageType {
	return []MessageType{
		MessageTypeReceipt,
		MessageTypeTyping,
		MessageTypeData,
		MessageTypeDataMessage,
		MessageTypeSync,
	}
}

// String returns the string representation of a message type.
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeReceipt:
		return "receipt"
	case MessageTypeTyping:
		return "typing"
	case MessageTypeData:
		return "data"
	case MessageTypeDataMessage:
		return "data-message"
	case MessageTypeSync:
		return "sync"
	case MessageTypeUnknown:
		fallthrough
	default:
		panic(fmt.Sprintf("unknown message type %d", mt))
	}
}

// ParseMessageType parses a message type given its representation as a string.
func ParseMessageType(mt string) (MessageType, error) {
	switch mt {
	case "receipt":
		return MessageTypeReceipt, nil
	case "typing":
		return MessageTypeTyping, nil
	case "data":
		return MessageTypeData, nil
	case "data-message":
		return MessageTypeDataMessage, nil
	case "sync":
		return MessageTypeSync, nil
	default:
		return MessageTypeUnknown, ErrMessageTypeUnknown
	}
}

// Message defines the message structure received from the Signal API.
type Message struct {
	Account  string   `json:"account"`
	Envelope Envelope `json:"envelope"`
}

// Envelope represents a message envelope.
type Envelope struct {
	Source         string          `json:"source"`
	SourceNumber   string          `json:"sourceNumber"`
	SourceUUID     string          `json:"sourceUuid"`
	SourceName     string          `json:"sourceName"`
	SourceDevice   int             `json:"sourceDevice"`
	Timestamp      int64           `json:"timestamp"`
	ReceiptMessage *ReceiptMessage `json:"receiptMessage,omitempty"`
	TypingMessage  *TypingMessage  `json:"typingMessage,omitempty"`
	DataMessage    *DataMessage    `json:"dataMessage,omitempty"`
	SyncMessage    *struct{}       `json:"syncMessage,omitempty"`
}

// ReceiptMessage represents a receipt message.
type ReceiptMessage struct {
	When       int64   `json:"when"`
	IsDelivery bool    `json:"isDelivery"`
	IsRead     bool    `json:"isRead"`
	IsViewed   bool    `json:"isViewed"`
	Timestamps []int64 `json:"timestamps"`
}

// TypingMessage represents a typing message.
type TypingMessage struct {
	Action    string `json:"action"`
	Timestamp int64  `json:"timestamp"`
}

// DataMessage represents a data message.
type DataMessage struct {
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

// MessageTypes returns the types of a message.
func (m Message) MessageTypes() []MessageType {
	mts := make([]MessageType, 0)

	if m.Envelope.ReceiptMessage != nil {
		mts = append(mts, MessageTypeReceipt)
	}

	if m.Envelope.TypingMessage != nil {
		mts = append(mts, MessageTypeTyping)
	}

	if m.Envelope.DataMessage != nil {
		mts = append(mts, MessageTypeData)

		if m.Envelope.DataMessage.Message != nil {
			mts = append(mts, MessageTypeDataMessage)
		}
	}

	if m.Envelope.SyncMessage != nil {
		mts = append(mts, MessageTypeSync)
	}

	return mts
}

// MessageTypes returns the types of a message encoded as a string.
func (m Message) MessageTypesStrings() []string {
	mts := m.MessageTypes()

	ss := make([]string, 0, len(mts))

	for _, mt := range mts {
		ss = append(ss, mt.String())
	}

	return ss
}
