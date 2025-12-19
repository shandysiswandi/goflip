package event

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
)

type handlerFunc func(ctx context.Context, event entity.FailedTxEvent) error

func (h handlerFunc) Handle(ctx context.Context, event entity.FailedTxEvent) error {
	return h(ctx, event)
}

func TestReconciliationConsumerRetriesAndIdempotent(t *testing.T) {
	bus := NewBus(10)

	var attempts int32
	done := make(chan struct{})
	handler := handlerFunc(func(ctx context.Context, event entity.FailedTxEvent) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return errors.New("temporary failure")
		}
		select {
		case <-done:
		default:
			close(done)
		}
		return nil
	})

	consumer := NewReconciliationConsumer(bus, handler, ConsumerConfig{
		Workers:     1,
		MaxRetries:  2,
		BaseBackoff: time.Millisecond,
	})
	consumer.Start()

	event := entity.FailedTxEvent{EventID: "evt-1", UploadID: "upload-1"}
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatalf("publish duplicate: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handler")
	}

	if err := consumer.Stop(context.Background()); err != nil {
		t.Fatalf("stop consumer: %v", err)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
