package event

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
)

type Handler interface {
	Handle(ctx context.Context, event entity.FailedTxEvent) error
}

type ConsumerConfig struct {
	Workers     int
	MaxRetries  int
	BaseBackoff time.Duration
}

type ReconciliationConsumer struct {
	bus         *Bus
	handler     Handler
	workers     int
	maxRetries  int
	baseBackoff time.Duration
	seen        sync.Map
	wg          sync.WaitGroup
}

func NewReconciliationConsumer(bus *Bus, handler Handler, cfg ConsumerConfig) *ReconciliationConsumer {
	workers := cfg.Workers
	if workers < 1 {
		workers = 4
	}

	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	baseBackoff := cfg.BaseBackoff
	if baseBackoff <= 0 {
		baseBackoff = 100 * time.Millisecond
	}

	return &ReconciliationConsumer{
		bus:         bus,
		handler:     handler,
		workers:     workers,
		maxRetries:  maxRetries,
		baseBackoff: baseBackoff,
	}
}

func (c *ReconciliationConsumer) Start() {
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.worker()
	}
}

func (c *ReconciliationConsumer) Stop(ctx context.Context) error {
	if c.bus != nil {
		c.bus.Close()
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ReconciliationConsumer) worker() {
	defer c.wg.Done()

	for event := range c.bus.Subscribe() {
		c.processEvent(event)
	}
}

func (c *ReconciliationConsumer) processEvent(event entity.FailedTxEvent) {
	if c.handler == nil {
		return
	}

	if event.EventID != "" {
		if _, loaded := c.seen.LoadOrStore(event.EventID, struct{}{}); loaded {
			slog.Info("skip duplicate failed transaction event", "event_id", event.EventID, "upload_id", event.UploadID)
			return
		}
	}

	backoff := c.baseBackoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		err := c.handler.Handle(context.Background(), event)
		if err == nil {
			return
		}

		if attempt == c.maxRetries {
			slog.Error("failed to reconcile transaction after retries", "event_id", event.EventID, "upload_id", event.UploadID, "error", err)
			return
		}

		if !sleepBackoff(backoff) {
			return
		}
		backoff *= 2
	}
}

func sleepBackoff(d time.Duration) bool {
	if d <= 0 {
		return false
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	<-timer.C
	return true
}

type NoopReconciler struct{}

func (NoopReconciler) Handle(ctx context.Context, event entity.FailedTxEvent) error {
	if event.EventID == "" {
		return errors.New("missing event id")
	}

	slog.Info("reconciled failed transaction", "event_id", event.EventID, "upload_id", event.UploadID)
	return nil
}
