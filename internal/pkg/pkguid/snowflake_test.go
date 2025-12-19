package pkguid

import "testing"

func TestGenerateRandomNodeIDRange(t *testing.T) {
	id, err := generateRandomNodeID()
	if err != nil {
		t.Fatalf("generateRandomNodeID: %v", err)
	}
	if id < 0 || id > 1023 {
		t.Fatalf("expected id within 0..1023, got %d", id)
	}
}

func TestSnowflakeGenerateUnique(t *testing.T) {
	gen, err := NewSnowflake()
	if err != nil {
		t.Fatalf("NewSnowflake: %v", err)
	}
	id1 := gen.Generate()
	id2 := gen.Generate()
	if id1 == id2 {
		t.Fatalf("expected unique ids, got %d and %d", id1, id2)
	}
}
