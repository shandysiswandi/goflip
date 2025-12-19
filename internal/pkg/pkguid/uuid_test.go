package pkguid

import (
	"testing"

	"github.com/google/uuid"
)

func TestUUIDGenerate(t *testing.T) {
	gen := NewUUID()
	id := gen.Generate()
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("expected valid uuid, got %q", id)
	}
}
