package pkglog

import (
	"context"
	"testing"
)

func TestCorrelationID(t *testing.T) {
	ctx := context.Background()
	if got := GetCorrelationID(ctx); got != "[invalid_chain_id]" {
		t.Fatalf("expected invalid chain id, got %q", got)
	}

	ctx = SetCorrelationID(ctx, "cid-123")
	if got := GetCorrelationID(ctx); got != "cid-123" {
		t.Fatalf("expected cid-123, got %q", got)
	}
}
