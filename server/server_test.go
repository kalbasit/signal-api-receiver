package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kalbasit/signal-api-receiver/receiver"
	"github.com/kalbasit/signal-api-receiver/server"
)

type mockClient struct {
	msgs []receiver.Message
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

			mc := &mockClient{msgs: []receiver.Message{}}

			s := server.New(mc)

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

			mc := &mockClient{msgs: []receiver.Message{}}

			s := server.New(mc)

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

			mc := &mockClient{msgs: []receiver.Message{}}

			s := server.New(mc)

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
	})

	t.Run("GET /receive/flush", func(t *testing.T) {
		t.Parallel()

		t.Run("no messages in the queue", func(t *testing.T) {
			t.Parallel()

			mc := &mockClient{msgs: []receiver.Message{}}

			s := server.New(mc)

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

			mc := &mockClient{msgs: []receiver.Message{}}

			s := server.New(mc)

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

			mc := &mockClient{msgs: []receiver.Message{}}

			s := server.New(mc)

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

				mc := &mockClient{msgs: []receiver.Message{}}

				s := server.New(mc)

				hs := httptest.NewServer(s)
				defer hs.Close()

				r, err := http.NewRequestWithContext(context.Background(), verb, hs.URL, nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(r)
				require.NoError(t, err)

				assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			})

			t.Run(verb+" /receive/flush", func(t *testing.T) {
				t.Parallel()

				mc := &mockClient{msgs: []receiver.Message{}}

				s := server.New(mc)

				hs := httptest.NewServer(s)
				defer hs.Close()

				r, err := http.NewRequestWithContext(context.Background(), verb, hs.URL+"/receive/flush", nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(r)
				require.NoError(t, err)

				assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			})

			t.Run(verb+" /receive/pop", func(t *testing.T) {
				t.Parallel()

				mc := &mockClient{msgs: []receiver.Message{}}

				s := server.New(mc)

				hs := httptest.NewServer(s)
				defer hs.Close()

				r, err := http.NewRequestWithContext(context.Background(), verb, hs.URL+"/receive/pop", nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(r)
				require.NoError(t, err)

				assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			})
		}
	})
}
