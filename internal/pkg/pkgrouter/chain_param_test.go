package pkgrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestChainOrder(t *testing.T) {
	order := make([]string, 0, 3)

	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), mw("mw1"), mw("mw2"))

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if !reflect.DeepEqual(order, []string{"mw1", "mw2", "handler"}) {
		t.Fatalf("unexpected order: %#v", order)
	}
}

func TestGetParam(t *testing.T) {
	params := httprouter.Params{{Key: "id", Value: "123"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)

	if got := GetParam(ctx, "id"); got != "123" {
		t.Fatalf("expected id=123, got %q", got)
	}
}
