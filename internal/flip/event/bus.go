package event

import (
	"context"
	"errors"
	"sync"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
)

var ErrBusClosed = errors.New("event bus is closed")

type Bus struct {
	mu     sync.RWMutex
	closed bool
	ch     chan entity.FailedTxEvent
}

func NewBus(buffer int) *Bus {
	if buffer < 1 {
		buffer = 1
	}

	return &Bus{
		ch: make(chan entity.FailedTxEvent, buffer),
	}
}

func (b *Bus) Publish(ctx context.Context, event entity.FailedTxEvent) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}

	select {
	case b.ch <- event:
		b.mu.RUnlock()
		return nil
	case <-ctx.Done():
		b.mu.RUnlock()
		return ctx.Err()
	}
}

func (b *Bus) Subscribe() <-chan entity.FailedTxEvent {
	return b.ch
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	close(b.ch)
}
