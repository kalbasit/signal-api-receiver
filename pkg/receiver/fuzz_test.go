package receiver_test

import (
	"encoding/json"
	"testing"

	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
)

func FuzzParseMessageType(f *testing.F) {
	for _, mt := range receiver.AllMessageTypes() {
		f.Add(mt.String())
	}

	f.Add("unknown")
	f.Add("")

	f.Fuzz(func(_ *testing.T, data string) {
		_, _ = receiver.ParseMessageType(data)
	})
}

func FuzzUnmarshalMessage(f *testing.F) {
	seed := []byte(`{
			"account": "+1234567890",
			"envelope": {
				"source": "+0987654321",
				"sourceNumber": "+0987654321",
				"sourceUuid": "uuid",
				"sourceName": "name",
				"sourceDevice": 1,
				"timestamp": 123456789,
				"dataMessage": {
					"timestamp": 123456789,
					"message": "hello"
				}
			}
		}`)
	f.Add(seed)

	f.Fuzz(func(t *testing.T, data []byte) {
		var m receiver.Message
		if err := json.Unmarshal(data, &m); err != nil {
			return
		}

		// If unmarshaling succeeds, we can try to call some methods on it
		_ = m.MessageTypes()
		_ = m.MessageTypesStrings()

		// Also check for potential panics in String() if we were to use it
		for _, mt := range m.MessageTypes() {
			func(mt receiver.MessageType) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panic in mt.String() for message type %d: %v", mt, r)
					}
				}()

				_ = mt.String()
			}(mt)
		}
	})
}
