package pkgroutine

import (
	"context"
	"errors"
	"testing"
)

func TestNewManagerDefaultMax(t *testing.T) {
	mgr := NewManager(0)
	if got := cap(mgr.sema); got != DefaultMaxGoroutine {
		t.Fatalf("expected cap %d, got %d", DefaultMaxGoroutine, got)
	}
}

func TestManagerCollectsErrors(t *testing.T) {
	mgr := NewManager(2)
	errOne := errors.New("one")
	errTwo := errors.New("two")

	mgr.Go(context.Background(), func(ctx context.Context) error {
		return errOne
	})
	mgr.Go(context.Background(), func(ctx context.Context) error {
		return errTwo
	})

	joined := mgr.Wait()
	if joined == nil {
		t.Fatalf("expected errors")
	}
	if !errors.Is(joined, errOne) {
		t.Fatalf("expected errOne to be present")
	}
	if !errors.Is(joined, errTwo) {
		t.Fatalf("expected errTwo to be present")
	}
}

func TestManagerRecoversPanics(t *testing.T) {
	mgr := NewManager(1)
	mgr.Go(context.Background(), func(ctx context.Context) error {
		panic("boom")
	})

	if err := mgr.Wait(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
