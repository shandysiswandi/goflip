package store

import (
	"context"
	"sync"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
	"github.com/shandysiswandi/goflip/internal/flip/usecase"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgerror"
)

type InMemoryStore struct {
	mu      sync.RWMutex
	uploads map[string]*uploadRecord
}

type uploadRecord struct {
	mu      sync.RWMutex
	meta    entity.UploadMeta
	balance int64
	issues  []entity.Transaction
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		uploads: make(map[string]*uploadRecord),
	}
}

func (s *InMemoryStore) CreateUpload(ctx context.Context, meta entity.UploadMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.uploads[meta.ID]; exists {
		return pkgerror.NewBusiness("upload already exists", pkgerror.CodeConflict)
	}

	s.uploads[meta.ID] = &uploadRecord{
		meta: meta,
	}

	return nil
}

func (s *InMemoryStore) UpdateMeta(ctx context.Context, uploadID string, fn func(meta *entity.UploadMeta)) error {
	rec, err := s.get(uploadID)
	if err != nil {
		return err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()

	fn(&rec.meta)

	return nil
}

func (s *InMemoryStore) SaveResults(ctx context.Context, uploadID string, balance int64, issues []entity.Transaction, totalLines, parsedOK, parseErr int64) error {
	rec, err := s.get(uploadID)
	if err != nil {
		return err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()

	rec.balance = balance
	rec.issues = issues
	rec.meta.TotalLines = totalLines
	rec.meta.ParsedOK = parsedOK
	rec.meta.ParseErr = parseErr

	return nil
}

func (s *InMemoryStore) GetBalance(ctx context.Context, uploadID string) (int64, entity.UploadMeta, error) {
	rec, err := s.get(uploadID)
	if err != nil {
		return 0, entity.UploadMeta{}, err
	}

	rec.mu.RLock()
	defer rec.mu.RUnlock()

	return rec.balance, rec.meta, nil
}

func (s *InMemoryStore) ListIssues(ctx context.Context, uploadID string, filter usecase.IssueFilter, page, pageSize int) ([]entity.Transaction, int, entity.UploadMeta, error) {
	rec, err := s.get(uploadID)
	if err != nil {
		return nil, 0, entity.UploadMeta{}, err
	}

	rec.mu.RLock()
	defer rec.mu.RUnlock()

	total := 0
	start := (page - 1) * pageSize
	end := start + pageSize
	items := make([]entity.Transaction, 0, pageSize)

	for _, tx := range rec.issues {
		if !filter.Matches(tx) {
			continue
		}

		if total >= start && total < end {
			items = append(items, tx)
		}
		total++
	}

	return items, total, rec.meta, nil
}

func (s *InMemoryStore) get(uploadID string) (*uploadRecord, error) {
	s.mu.RLock()
	rec, ok := s.uploads[uploadID]
	s.mu.RUnlock()
	if !ok {
		return nil, pkgerror.ErrNotFound
	}

	return rec, nil
}
