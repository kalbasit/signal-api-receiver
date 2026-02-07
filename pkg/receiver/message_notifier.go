package receiver

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/zerolog"
)

// ErrMessageNotifierClosed is returned when handlers are no longer accepting work.
var ErrMessageNotifierClosed = errors.New("notifier is closed")

type MessageNotifierPayload struct {
	Message Message
}

type MessageNotifierTrigger func(ctx context.Context, payload MessageNotifierPayload) error

type handleable interface {
	Handle(context.Context, MessageNotifierPayload) error
}

type MessageNotifier struct {
	logger   zerolog.Logger
	sliceMu  sync.RWMutex
	runMu    sync.RWMutex
	wg       sync.WaitGroup
	closed   bool
	handlers []handleable
}

func InitNotifier(ctx context.Context) (*MessageNotifier, MessageNotifierTrigger) {
	notifier := MessageNotifier{
		logger:   *zerolog.Ctx(ctx),
		closed:   false,
		handlers: make([]handleable, 0),
	}

	return &notifier, notifier.trigger
}

func (u *MessageNotifier) RegisterHandler(_ context.Context, handler handleable) {
	u.sliceMu.Lock()
	defer u.sliceMu.Unlock()

	u.handlers = append(u.handlers, handler)
}

func (u *MessageNotifier) Shutdown(ctx context.Context) error {
	u.logger.Debug().Msg("Closing notifier pipeline")
	u.runMu.Lock()
	u.closed = true
	u.runMu.Unlock()

	done := make(chan struct{})

	go func() {
		u.logger.Debug().Msg("Waiting for active handlers to complete")
		u.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (u *MessageNotifier) trigger(ctx context.Context, payload MessageNotifierPayload) error {
	if len(u.handlers) == 0 {
		return nil
	}

	u.runMu.RLock()
	// Keep trigger from racing with shutdown
	if u.closed {
		u.logger.Debug().Msg("Notifier pipeline was closed. Skip handler execution")
		u.runMu.RUnlock()

		return ErrMessageNotifierClosed
	}

	u.sliceMu.RLock()
	// Copy handlers to prevent modification during launch iteration.
	handlers := make([]handleable, len(u.handlers))
	copy(handlers, u.handlers)
	u.sliceMu.RUnlock()

	for _, handler := range handlers {
		if err := ctx.Err(); err != nil {
			// Context errors interrupt handler launching while keep handlers in-flight
			u.logger.Debug().Msg("Skip remaining notifier handlers")
			u.runMu.RUnlock()

			return err
		}

		handler := handler

		u.wg.Add(1)

		go func() {
			defer u.wg.Done()

			if err := handler.Handle(ctx, payload); err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("error while handling new-message")
			}
		}()
	}

	u.runMu.RUnlock()

	return nil
}
