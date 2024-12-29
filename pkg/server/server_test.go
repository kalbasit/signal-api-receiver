package server_test

import (
	"context"
	"encoding/json"
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

	"github.com/kalbasit/signal-api-receiver/pkg/receiver"
	"github.com/kalbasit/signal-api-receiver/pkg/server"
)

type mockClient struct {
	connectCalled int
	connectErr    chan error
	recvMsg       chan receiver.Message
	recvErr       chan error

	msgs []receiver.Message
}

func newMockClient() *mockClient {
	return &mockClient{
		connectErr: make(chan error),
		recvMsg:    make(chan receiver.Message),
		recvErr:    make(chan error),

		msgs: []receiver.Message{},
	}
}

func (mc *mockClient) Connect() error {
	mc.connectCalled++
	if mc.connectErr != nil {
		return <-mc.connectErr
	}

	return nil
}

func (mc *mockClient) ReceiveLoop() error {
	for {
		select {
		case msg := <-mc.recvMsg:
			mc.msgs = append(mc.msgs, msg)
		case err := <-mc.recvErr:
			return err
		}
	}
}

func (mc *mockClient) Pop() *receiver.Message {
	if len(mc.msgs) == 0 {
		return nil
	}

	msg := mc.msgs[0]
	mc.msgs = mc.msgs[1:]

	return &msg
}

func (mc *mockClient) Flush() []receiver.Message {
	msgs := mc.msgs
	mc.msgs = []receiver.Message{}

	return msgs
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("GET /receive/pop", func(t *testing.T) {
		t.Parallel()

		t.Run("no messages in the queue", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, false)

			hs := httptest.NewServer(s)
			defer hs.Close()

			mc.msgs = []receiver.Message{}

			//nolint:noctx
			resp, err := http.Get(hs.URL + "/receive/pop")
			require.NoError(t, err)

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("one message in the queue", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, false)

			hs := httptest.NewServer(s)
			defer hs.Close()

			want := receiver.Message{Account: "0"}
			mc.msgs = []receiver.Message{want}

			//nolint:noctx
			resp, err := http.Get(hs.URL + "/receive/pop")
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var got receiver.Message

			require.NoError(t, json.Unmarshal(body, &got))

			assert.Equal(t, want, got)
		})

		t.Run("three messages in the queue", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, false)

			hs := httptest.NewServer(s)
			defer hs.Close()

			want := []receiver.Message{
				{Account: "0"},
				{Account: "1"},
				{Account: "2"},
			}
			mc.msgs = want

			var got []receiver.Message

			for range want {
				//nolint:noctx
				resp, err := http.Get(hs.URL + "/receive/pop")
				require.NoError(t, err)

				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var m receiver.Message

				require.NoError(t, json.Unmarshal(body, &m))

				got = append(got, m)
			}

			assert.Equal(t, want, got)
		})

		t.Run("can repeat last message if enabled", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, true)

			hs := httptest.NewServer(s)
			defer hs.Close()

			want := receiver.Message{Account: "0"}
			mc.msgs = []receiver.Message{want}

			for i := 0; i < 3; i++ {
				//nolint:noctx
				resp, err := http.Get(hs.URL + "/receive/pop")
				require.NoError(t, err)

				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var got receiver.Message

				require.NoError(t, json.Unmarshal(body, &got))

				assert.Equal(t, want, got)
			}
		})
	})

	t.Run("GET /receive/flush", func(t *testing.T) {
		t.Parallel()

		t.Run("no messages in the queue", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, false)

			hs := httptest.NewServer(s)
			defer hs.Close()

			mc.msgs = []receiver.Message{}

			//nolint:noctx
			resp, err := http.Get(hs.URL + "/receive/flush")
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var got []receiver.Message

			require.NoError(t, json.Unmarshal(body, &got))

			assert.Empty(t, got)
		})

		t.Run("one message in the queue", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, false)

			hs := httptest.NewServer(s)
			defer hs.Close()

			want := []receiver.Message{{Account: "0"}}
			mc.msgs = want

			//nolint:noctx
			resp, err := http.Get(hs.URL + "/receive/flush")
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var got []receiver.Message

			require.NoError(t, json.Unmarshal(body, &got))

			assert.Equal(t, want, got)
		})

		t.Run("three messages in the queue", func(t *testing.T) {
			t.Parallel()

			mc := newMockClient()

			s := server.New(newContext(), mc, false)

			hs := httptest.NewServer(s)
			defer hs.Close()

			want := []receiver.Message{
				{Account: "0"},
				{Account: "1"},
				{Account: "2"},
			}
			mc.msgs = want

			//nolint:noctx
			resp, err := http.Get(hs.URL + "/receive/flush")
			require.NoError(t, err)

			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var got []receiver.Message

			require.NoError(t, json.Unmarshal(body, &got))

			assert.Equal(t, want, got)
		})
	})

	t.Run("anything else", func(t *testing.T) {
		t.Parallel()

		for _, verb := range []string{"POST", "PUT", "PATCH", "DELETE"} {
			t.Run(verb+" /", func(t *testing.T) {
				t.Parallel()

				mc := newMockClient()

				s := server.New(newContext(), mc, false)

				hs := httptest.NewServer(s)
				defer hs.Close()

				r, err := http.NewRequestWithContext(context.Background(), verb, hs.URL, nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(r)
				require.NoError(t, err)

				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			})

			t.Run(verb+" /receive/flush", func(t *testing.T) {
				t.Parallel()

				mc := newMockClient()

				s := server.New(newContext(), mc, false)

				hs := httptest.NewServer(s)
				defer hs.Close()

				r, err := http.NewRequestWithContext(context.Background(), verb, hs.URL+"/receive/flush", nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(r)
				require.NoError(t, err)

				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})

			t.Run(verb+" /receive/pop", func(t *testing.T) {
				t.Parallel()

				mc := newMockClient()

				s := server.New(newContext(), mc, false)

				hs := httptest.NewServer(s)
				defer hs.Close()

				r, err := http.NewRequestWithContext(context.Background(), verb, hs.URL+"/receive/pop", nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(r)
				require.NoError(t, err)

				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		}
	})
}

func TestServerReconnect(t *testing.T) {
	t.Parallel()

	mc := newMockClient()

	server.New(newContext(), mc, false)

	assert.Zero(t, mc.connectCalled)

	mc.recvMsg <- receiver.Message{Account: "0"}

	mc.recvErr <- nil
	mc.connectErr <- nil

	assert.Len(t, mc.msgs, 1)

	mc.recvMsg <- receiver.Message{Account: "0"}

	assert.Equal(t, 1, mc.connectCalled)

	mc.recvErr <- nil

	assert.Len(t, mc.msgs, 2)
}

func TestConnectAndReconnect(t *testing.T) {
	t.Parallel()

	newMessage := func(i int) receiver.Message {
		message := strconv.Itoa(i)

		return receiver.Message{
			Envelope: receiver.Envelope{
				DataMessage: &receiver.DataMessage{
					Message: &message,
				},
			},
		}
	}

	var sentC chan struct{}

	ch := make(chan chan receiver.Message, 1)
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

		close(sentC)
	}))

	defer trs.Close()

	uri, err := url.Parse(trs.URL)
	require.NoError(t, err)

	uri.Scheme = "ws"

	client, err := receiver.New(newContext(), uri, receiver.MessageTypeDataMessage.String())
	require.NoError(t, err)

	s := server.New(newContext(), client, true)

	tss := httptest.NewServer(s)
	defer tss.Close()

	for i := 1; i <= 2; i++ {
		sentC = make(chan struct{})
		messages := make(chan receiver.Message, 3)

		for j := 1; j <= 3; j++ {
			messages <- newMessage(i * j)
		}

		close(messages)
		ch <- messages

		// wait for the receiver to read all messages
		<-sentC

		for j := 1; j <= 3; j++ {
			r, err := http.NewRequestWithContext(newContext(), http.MethodGet, tss.URL+"/receive/pop", nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(r)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, resp.StatusCode)

			defer func() {
				//nolint:errcheck
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}()

			var msg receiver.Message

			require.NoError(t, json.NewDecoder(resp.Body).Decode(&msg))

			assert.Equal(t, strconv.Itoa(i*j), *msg.Envelope.DataMessage.Message)
		}
	}
}

func newContext() context.Context {
	return zerolog.
		New(io.Discard).
		WithContext(context.Background())
}
