package pkgconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestViperConfigValues(t *testing.T) {
	path := writeConfigFile(t, "int: 42\nbool: true\nfloat: 3.14\nstring: hi\nbinary: aGVsbG8=\narray: a,b,c\nmap: k1:v1,k2:v2\n")

	cfg, err := NewViper(path)
	if err != nil {
		t.Fatalf("NewViper: %v", err)
	}
	defer func() {
		if err := cfg.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	if got := cfg.GetInt("int"); got != 42 {
		t.Fatalf("GetInt: expected 42, got %d", got)
	}
	if got := cfg.GetBool("bool"); got != true {
		t.Fatalf("GetBool: expected true, got %v", got)
	}
	if got := cfg.GetFloat("float"); got != 3.14 {
		t.Fatalf("GetFloat: expected 3.14, got %v", got)
	}
	if got := cfg.GetString("string"); got != "hi" {
		t.Fatalf("GetString: expected hi, got %q", got)
	}
	if got := string(cfg.GetBinary("binary")); got != "hello" {
		t.Fatalf("GetBinary: expected hello, got %q", got)
	}
	if got := cfg.GetArray("array"); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("GetArray: unexpected value: %#v", got)
	}
	if got := cfg.GetMap("map"); !reflect.DeepEqual(got, map[string]string{"k1": "v1", "k2": "v2"}) {
		t.Fatalf("GetMap: unexpected value: %#v", got)
	}
}

func TestViperGetBinaryInvalid(t *testing.T) {
	path := writeConfigFile(t, "binary: not-base64\n")
	cfg, err := NewViper(path)
	if err != nil {
		t.Fatalf("NewViper: %v", err)
	}

	if got := cfg.GetBinary("binary"); got != nil {
		t.Fatalf("expected nil for invalid base64, got %v", got)
	}
}
