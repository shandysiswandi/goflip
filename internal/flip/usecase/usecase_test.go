package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgerror"
)

type testStore struct {
	mu      sync.RWMutex
	metas   map[string]entity.UploadMeta
	balance map[string]int64
	issues  map[string][]entity.Transaction
}

func newTestStore() *testStore {
	return &testStore{
		metas:   make(map[string]entity.UploadMeta),
		balance: make(map[string]int64),
		issues:  make(map[string][]entity.Transaction),
	}
}

func (s *testStore) CreateUpload(ctx context.Context, meta entity.UploadMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metas[meta.ID] = meta
	return nil
}

func (s *testStore) UpdateMeta(ctx context.Context, uploadID string, fn func(meta *entity.UploadMeta)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.metas[uploadID]
	if !ok {
		return pkgerror.ErrNotFound
	}
	fn(&meta)
	s.metas[uploadID] = meta
	return nil
}

func (s *testStore) SaveResults(ctx context.Context, uploadID string, balance int64, issues []entity.Transaction, totalLines, parsedOK, parseErr int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.metas[uploadID]; !ok {
		return pkgerror.ErrNotFound
	}
	s.balance[uploadID] = balance
	s.issues[uploadID] = issues
	return nil
}

func (s *testStore) GetBalance(ctx context.Context, uploadID string) (int64, entity.UploadMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	meta, ok := s.metas[uploadID]
	if !ok {
		return 0, entity.UploadMeta{}, pkgerror.ErrNotFound
	}
	return s.balance[uploadID], meta, nil
}

func (s *testStore) ListIssues(ctx context.Context, uploadID string, filter IssueFilter, page, pageSize int) ([]entity.Transaction, int, entity.UploadMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	meta, ok := s.metas[uploadID]
	if !ok {
		return nil, 0, entity.UploadMeta{}, pkgerror.ErrNotFound
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	total := 0
	issues := make([]entity.Transaction, 0, pageSize)
	for _, tx := range s.issues[uploadID] {
		if !filter.Matches(tx) {
			continue
		}
		if total >= start && total < end {
			issues = append(issues, tx)
		}
		total++
	}

	return issues, total, meta, nil
}

type testPublisher struct {
	mu     sync.Mutex
	events []entity.FailedTxEvent
}

func (p *testPublisher) Publish(ctx context.Context, event entity.FailedTxEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

type testID struct {
	mu sync.Mutex
	n  int
}

func (t *testID) Generate() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n++
	return fmt.Sprintf("id-%d", t.n)
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time {
	return f.now
}

func TestProcessUploadComputesBalanceAndIssues(t *testing.T) {
	store := newTestStore()
	events := &testPublisher{}
	clock := fixedClock{now: time.Unix(123, 0)}
	ids := &testID{}

	uc := &Usecase{
		store:   store,
		events:  events,
		clock:   clock,
		id:      ids,
		rootCtx: context.Background(),
	}

	uploadID := "upload-1"
	if err := store.CreateUpload(context.Background(), entity.UploadMeta{ID: uploadID}); err != nil {
		t.Fatalf("create upload: %v", err)
	}

	csv := strings.Join([]string{
		"1674507883, JOHN DOE, CREDIT, 100, SUCCESS, salary",
		"1674507884, JOHN DOE, DEBIT, 50, SUCCESS, grocery",
		"1674507885, JOHN DOE, DEBIT, 20, FAILED, restaurant",
		"1674507886, JOHN DOE, CREDIT, 10, PENDING, transfer",
	}, "\n")

	if err := uc.processUpload(context.Background(), uploadID, strings.NewReader(csv)); err != nil {
		t.Fatalf("process upload: %v", err)
	}

	balance, meta, err := store.GetBalance(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if balance != 50 {
		t.Fatalf("unexpected balance: %d", balance)
	}
	if meta.Status != entity.UploadStatusDone {
		t.Fatalf("expected status done, got %s", meta.Status)
	}
	if meta.TotalLines != 4 || meta.ParsedOK != 4 || meta.ParseErr != 0 {
		t.Fatalf("unexpected stats: %+v", meta)
	}

	issues, total, _, err := store.ListIssues(context.Background(), uploadID, IssueFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if total != 2 || len(issues) != 2 {
		t.Fatalf("expected 2 issues, got total=%d len=%d", total, len(issues))
	}
	if len(events.events) != 1 {
		t.Fatalf("expected 1 failed event, got %d", len(events.events))
	}
}

func TestProcessUploadCountsParseErrors(t *testing.T) {
	store := newTestStore()
	clock := fixedClock{now: time.Unix(456, 0)}
	ids := &testID{}

	uc := &Usecase{
		store:   store,
		clock:   clock,
		id:      ids,
		rootCtx: context.Background(),
	}

	uploadID := "upload-2"
	if err := store.CreateUpload(context.Background(), entity.UploadMeta{ID: uploadID}); err != nil {
		t.Fatalf("create upload: %v", err)
	}

	csv := strings.Join([]string{
		"1674507883, JOHN DOE, CREDIT, 100, SUCCESS, salary",
		"invalid,line",
		"1674507884, JOHN DOE, DEBIT, 50, SUCCESS, grocery",
	}, "\n")

	if err := uc.processUpload(context.Background(), uploadID, strings.NewReader(csv)); err != nil {
		t.Fatalf("process upload: %v", err)
	}

	_, meta, err := store.GetBalance(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if meta.TotalLines != 3 || meta.ParsedOK != 2 || meta.ParseErr != 1 {
		t.Fatalf("unexpected stats: %+v", meta)
	}
}
