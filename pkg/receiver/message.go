package receiver

// Message defines the message structure received from the Signal API.
type Message struct {
	Envelope struct {
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
	} `json:"envelope"`

	Account string `json:"account"`
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
