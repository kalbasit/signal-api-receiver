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

		assert.Equal(t, receiver.MessageTypeReceiptMessage, m.MessageType())
	})

	t.Run("MessageTypeTypingMessage", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				TypingMessage: &receiver.TypingMessage{},
			},
		}

		assert.Equal(t, receiver.MessageTypeTypingMessage, m.MessageType())
	})

	t.Run("MessageTypeDataMessage", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				DataMessage: &receiver.DataMessage{},
			},
		}

		assert.Equal(t, receiver.MessageTypeDataMessage, m.MessageType())
	})

	t.Run("MessageTypeSyncMessage", func(t *testing.T) {
		t.Parallel()

		m := receiver.Message{
			Envelope: receiver.Envelope{
				SyncMessage: &struct{}{},
			},
		}

		assert.Equal(t, receiver.MessageTypeSyncMessage, m.MessageType())
	})
}
