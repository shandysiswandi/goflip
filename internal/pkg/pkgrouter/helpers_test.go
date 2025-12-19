package pkgrouter

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeCID(t *testing.T) {
	if got := normalizeCID("  abc  "); got != "abc" {
		t.Fatalf("expected trimmed value, got %q", got)
	}
	if got := normalizeCID("\n"); got != "" {
		t.Fatalf("expected empty for newline, got %q", got)
	}
	long := strings.Repeat("a", 200)
	if got := normalizeCID(long); len(got) != 128 {
		t.Fatalf("expected length 128, got %d", len(got))
	}
}

func TestMaskHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "secret")
	headers.Set("X-Trace", "ok")

	masked := maskHeaders(headers)
	if got := masked.Get("Authorization"); got != "***" {
		t.Fatalf("expected masked authorization, got %q", got)
	}
	if got := masked.Get("X-Trace"); got != "ok" {
		t.Fatalf("expected X-Trace to stay, got %q", got)
	}
	if got := headers.Get("Authorization"); got != "secret" {
		t.Fatalf("expected original headers unchanged, got %q", got)
	}
}

func TestMaskData(t *testing.T) {
	input := map[string]any{
		"password": "secret",
		"profile": map[string]any{
			"access_token": "token",
		},
		"items": []any{
			map[string]any{
				"refresh_token": "rt",
			},
		},
	}

	masked := maskData(input).(map[string]any)
	if masked["password"] != "***" {
		t.Fatalf("expected masked password")
	}
	if masked["profile"].(map[string]any)["access_token"] != "***" {
		t.Fatalf("expected masked access_token")
	}
	items := masked["items"].([]any)
	if items[0].(map[string]any)["refresh_token"] != "***" {
		t.Fatalf("expected masked refresh_token")
	}
}

func TestParseAndMaskBodyJSON(t *testing.T) {
	body := []byte(`{"password":"secret","name":"bob"}`)
	parsed := parseAndMaskBody("application/json", body)

	m, ok := parsed.(map[string]any)
	if !ok {
		encoded, _ := json.Marshal(parsed)
		t.Fatalf("expected map, got %s", string(encoded))
	}
	if m["password"] != "***" {
		t.Fatalf("expected masked password")
	}
	if m["name"] != "bob" {
		t.Fatalf("expected name to remain")
	}
}

func TestParseAndMaskBodyForm(t *testing.T) {
	body := []byte("password=secret&name=bob")
	parsed := parseAndMaskBody("application/x-www-form-urlencoded", body)

	m, ok := parsed.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", parsed)
	}
	if m["password"] != "***" {
		t.Fatalf("expected masked password")
	}
	if m["name"] != "bob" {
		t.Fatalf("expected name to remain")
	}
}

func TestParseAndMaskBodyBinary(t *testing.T) {
	body := []byte{0xff, 0xfe, 0xfd}
	parsed := parseAndMaskBody("text/plain", body)
	if !reflect.DeepEqual(parsed, "<binary body omitted>") {
		t.Fatalf("expected binary body omission, got %v", parsed)
	}
}
