//nolint:testpackage
package receiver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals
var logger = zerolog.New(io.Discard)

func TestFlush(t *testing.T) {
	t.Parallel()

	t.Run("returns empty list when no messages was found", func(t *testing.T) {
		t.Parallel()

		c := &Client{logger: logger, messages: []Message{}}
		assert.Equal(t, []Message{}, c.Flush())
	})

	t.Run("return the message if only one is there", func(t *testing.T) {
		t.Parallel()

		c := &Client{logger: logger, messages: []Message{{Account: "1"}}}

		assert.Equal(t, []Message{{Account: "1"}}, c.Flush())
	})

	t.Run("return messages in order", func(t *testing.T) {
		t.Parallel()

		c := &Client{
			logger: logger,
			messages: []Message{
				{Account: "0"},
				{Account: "1"},
				{Account: "2"},
			},
		}

		want := []Message{
			{Account: "0"},
			{Account: "1"},
			{Account: "2"},
		}
		got := c.Flush()

		assert.Equal(t, want, got)
	})
}

func TestPop(t *testing.T) {
	t.Parallel()

	t.Run("returns null when no messages was found", func(t *testing.T) {
		t.Parallel()

		c := &Client{logger: logger, messages: []Message{}}

		var want *Message

		assert.Equal(t, want, c.Pop())
	})

	t.Run("return the message if only one is there", func(t *testing.T) {
		t.Parallel()

		c := &Client{logger: logger, messages: []Message{{Account: "1"}}}
		want := Message{Account: "1"}
		assert.Equal(t, want, *c.Pop())
	})

	t.Run("return messages in order", func(t *testing.T) {
		t.Parallel()

		c := &Client{
			logger: logger,
			messages: []Message{
				{Account: "0"},
				{Account: "1"},
				{Account: "2"},
			},
		}

		for i := range c.messages {
			want := Message{Account: strconv.Itoa(i)}
			assert.Equal(t, want, *c.Pop())
		}
	})
}

func TestRecordMessageTypes(t *testing.T) {
	t.Parallel()

	ch := make(chan chan Message)
	trs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)

			return
		}
		defer conn.Close()

		messages := <-ch
		for msg := range messages {
			if err := conn.WriteJSON(msg); err != nil {
				t.Errorf("write message: %v", err)

				return
			}
		}
	}))

	defer trs.Close()

	uri, err := url.Parse(trs.URL)
	require.NoError(t, err)

	uri.Scheme = "ws"

	client, err := New(newContext(), uri, MessageTypeDataMessage.String())
	require.NoError(t, err)

	go func(t *testing.T) {
		t.Helper()

		assert.NoError(t, client.ReceiveLoop())
	}(t)

	assert.Nil(t, client.Pop())
}

func newContext() context.Context {
	return zerolog.
		New(io.Discard).
		WithContext(context.Background())
}
