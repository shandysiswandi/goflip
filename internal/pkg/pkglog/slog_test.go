package pkglog

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type captureHandler struct {
	attrs map[string]slog.Value
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	if h.attrs == nil {
		h.attrs = make(map[string]slog.Value)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.attrs[a.Key] = a.Value
		return true
	})
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(_ string) slog.Handler {
	return h
}

func TestContextHandlerAddsServiceAndCID(t *testing.T) {
	capture := &captureHandler{}
	handler := &contextHandler{Handler: capture}

	ctx := SetCorrelationID(context.Background(), "cid-abc")
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)

	if err := handler.Handle(ctx, rec); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if got := capture.attrs["service"].String(); got != "goflip" {
		t.Fatalf("expected service=goflip, got %q", got)
	}
	if got := capture.attrs["_cID"].String(); got != "cid-abc" {
		t.Fatalf("expected _cID=cid-abc, got %q", got)
	}
}

func TestContextHandlerSkipsInvalidCID(t *testing.T) {
	capture := &captureHandler{}
	handler := &contextHandler{Handler: capture}

	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)

	if err := handler.Handle(ctx, rec); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if _, ok := capture.attrs["_cID"]; ok {
		t.Fatalf("did not expect _cID to be set")
	}
	if got := capture.attrs["service"].String(); got != "goflip" {
		t.Fatalf("expected service=goflip, got %q", got)
	}
}
