package receiver

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlush(t *testing.T) {
	t.Run("returns empty list when no messages was found", func(t *testing.T) {
		c := &Client{messages: []Message{}}
		assert.Equal(t, []Message{}, c.Flush())
	})

	t.Run("return the message if only one is there", func(t *testing.T) {
		c := &Client{messages: []Message{{Account: "1"}}}

		assert.Equal(t, []Message{{Account: "1"}}, c.Flush())
	})

	t.Run("return messages in order", func(t *testing.T) {
		c := &Client{messages: []Message{
			{Account: "0"},
			{Account: "1"},
			{Account: "2"},
		}}

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
	t.Run("returns null when no messages was found", func(t *testing.T) {
		c := &Client{messages: []Message{}}
		var want *Message
		assert.Equal(t, want, c.Pop())
	})

	t.Run("return the message if only one is there", func(t *testing.T) {
		c := &Client{messages: []Message{{Account: "1"}}}
		want := Message{Account: "1"}
		assert.Equal(t, want, *c.Pop())
	})

	t.Run("return messages in order", func(t *testing.T) {
		c := &Client{messages: []Message{
			{Account: "0"},
			{Account: "1"},
			{Account: "2"},
		}}

		for i := range c.messages {
			want := Message{Account: strconv.Itoa(i)}
			assert.Equal(t, want, *c.Pop())
		}
	})
}
