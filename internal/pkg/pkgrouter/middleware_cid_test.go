package pkgrouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shandysiswandi/goflip/internal/pkg/pkglog"
)

type staticGenerator struct {
	value string
	calls int
}

func (g *staticGenerator) Generate() string {
	g.calls++
	return g.value
}

func TestMiddlewareCorrelationIDUsesHeader(t *testing.T) {
	gen := &staticGenerator{value: "generated"}
	mw := middlewareCorrelationID(gen)

	var gotCID string
	wrapped := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCID = pkglog.GetCorrelationID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set(HeaderCorrelationID, "header-cid")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderCorrelationID); got != "header-cid" {
		t.Fatalf("expected response cid header, got %q", got)
	}
	if gotCID != "header-cid" {
		t.Fatalf("expected context cid header-cid, got %q", gotCID)
	}
	if gen.calls != 0 {
		t.Fatalf("expected generator not called")
	}
}

func TestMiddlewareCorrelationIDGeneratesWhenMissing(t *testing.T) {
	gen := &staticGenerator{value: "generated"}
	mw := middlewareCorrelationID(gen)

	var gotCID string
	wrapped := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCID = pkglog.GetCorrelationID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderCorrelationID); got != "generated" {
		t.Fatalf("expected response cid header, got %q", got)
	}
	if gotCID != "generated" {
		t.Fatalf("expected context cid generated, got %q", gotCID)
	}
	if gen.calls != 1 {
		t.Fatalf("expected generator called once")
	}
}
