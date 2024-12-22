package receiver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
)

func TestMessageType(t *testing.T) {
	t.Parallel()

	t.Run("MessageTypeReceiptMessage", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				ReceiptMessage: &receiver.ReceiptMessage{},
			},
		}

		assert.Equal(t,
			[]receiver.MessageType{receiver.MessageTypeReceipt},
			m.MessageTypes())
	})

	t.Run("MessageTypeTypingMessage", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				TypingMessage: &receiver.TypingMessage{},
			},
		}

		assert.Equal(t,
			[]receiver.MessageType{receiver.MessageTypeTyping},
			m.MessageTypes())
	})

	t.Run("MessageTypeData", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				DataMessage: &receiver.DataMessage{},
			},
		}

		assert.Equal(t,
			[]receiver.MessageType{receiver.MessageTypeData},
			m.MessageTypes())
	})

	t.Run("MessageTypeDataMessage", func(t *testing.T) {
		t.Parallel()

		msg := "test"

		m := receiver.Message{
			Envelope: receiver.Envelope{
				DataMessage: &receiver.DataMessage{Message: &msg},
			},
		}

		assert.Equal(t,
			[]receiver.MessageType{
				receiver.MessageTypeData,
				receiver.MessageTypeDataMessage,
			},
			m.MessageTypes())
	})

	t.Run("MessageTypeSyncMessage", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				SyncMessage: &struct{}{},
			},
		}

		assert.Equal(t,
			[]receiver.MessageType{receiver.MessageTypeSync},
			m.MessageTypes())
	})
}
