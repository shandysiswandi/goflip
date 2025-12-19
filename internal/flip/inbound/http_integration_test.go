package inbound

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
	"github.com/shandysiswandi/goflip/internal/flip/event"
	"github.com/shandysiswandi/goflip/internal/flip/store"
	"github.com/shandysiswandi/goflip/internal/flip/usecase"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgrouter"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgroutine"
	"github.com/shandysiswandi/goflip/internal/pkg/pkguid"
)

type envelope[T any] struct {
	Data T              `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

func TestUploadProcessQuery(t *testing.T) {
	runner := pkgroutine.NewManager(10)
	storage := store.NewInMemoryStore()
	bus := event.NewBus(10)

	uc := usecase.New(usecase.Dependency{
		Store:   storage,
		Events:  bus,
		Runner:  runner,
		ID:      pkguid.NewUUID(),
		RootCtx: context.Background(),
	})

	router := pkgrouter.NewRouter(pkguid.NewUUID())
	RegisterHTTPEndpoint(router, uc)

	uploadID := uploadCSV(t, router)

	var balance BalanceResponse
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		balance = getBalance(t, router, uploadID)
		if balance.Status == entity.UploadStatusDone {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if balance.Status != entity.UploadStatusDone {
		t.Fatalf("upload not done, status=%s", balance.Status)
	}
	if balance.Balance != 50 {
		t.Fatalf("unexpected balance: %d", balance.Balance)
	}

	issues := getIssues(t, router, uploadID)
	if len(issues.Transactions) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues.Transactions))
	}

	if err := runner.Wait(); err != nil {
		t.Fatalf("runner wait: %v", err)
	}
}

func uploadCSV(t *testing.T, router http.Handler) string {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "statement.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	csv := []byte("1674507883, JOHN DOE, CREDIT, 100, SUCCESS, salary\n" +
		"1674507884, JOHN DOE, DEBIT, 50, SUCCESS, grocery\n" +
		"1674507885, JOHN DOE, DEBIT, 20, FAILED, restaurant\n" +
		"1674507886, JOHN DOE, CREDIT, 10, PENDING, transfer\n")

	if _, err := part.Write(csv); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/statements", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var env envelope[UploadResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if env.Data.UploadID == "" {
		t.Fatal("upload id is empty")
	}

	return env.Data.UploadID
}

func getBalance(t *testing.T, router http.Handler, uploadID string) BalanceResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/balance?upload_id="+uploadID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected balance status: %d", rec.Code)
	}

	var env envelope[BalanceResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode balance: %v", err)
	}

	return env.Data
}

func getIssues(t *testing.T, router http.Handler, uploadID string) TransactionIssuesResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/transactions/issues?upload_id="+uploadID+"&page=1&page_size=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected issues status: %d", rec.Code)
	}

	var env envelope[TransactionIssuesResponse]
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode issues: %v", err)
	}

	return env.Data
}
