package store

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
	"github.com/shandysiswandi/goflip/internal/flip/usecase"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgerror"
)

func TestInMemoryStore_CreateUpload_Duplicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewInMemoryStore()
	meta := entity.UploadMeta{
		ID:        "upload-1",
		Status:    entity.UploadStatusQueued,
		StartedAt: 100,
	}

	if err := store.CreateUpload(ctx, meta); err != nil {
		t.Fatalf("CreateUpload() err = %v", err)
	}

	err := store.CreateUpload(ctx, meta)
	if err == nil {
		t.Fatal("CreateUpload() expected error, got nil")
	}

	var perr *pkgerror.Error
	if !errors.As(err, &perr) {
		t.Fatalf("CreateUpload() expected pkgerror.Error, got %T", err)
	}

	if perr.Code() != pkgerror.CodeConflict {
		t.Fatalf("CreateUpload() error code = %v, want %v", perr.Code(), pkgerror.CodeConflict)
	}
}

func TestInMemoryStore_UpdateMeta_And_GetBalance(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewInMemoryStore()
	meta := entity.UploadMeta{
		ID:        "upload-2",
		Status:    entity.UploadStatusQueued,
		StartedAt: 123,
	}

	if err := store.CreateUpload(ctx, meta); err != nil {
		t.Fatalf("CreateUpload() err = %v", err)
	}

	err := store.UpdateMeta(ctx, meta.ID, func(m *entity.UploadMeta) {
		m.Status = entity.UploadStatusDone
		m.Err = "none"
		m.EndedAt = 456
	})
	if err != nil {
		t.Fatalf("UpdateMeta() err = %v", err)
	}

	balance, gotMeta, err := store.GetBalance(ctx, meta.ID)
	if err != nil {
		t.Fatalf("GetBalance() err = %v", err)
	}

	if balance != 0 {
		t.Fatalf("GetBalance() balance = %d, want 0", balance)
	}

	if gotMeta.Status != entity.UploadStatusDone {
		t.Fatalf("GetBalance() meta status = %v, want %v", gotMeta.Status, entity.UploadStatusDone)
	}

	if gotMeta.Err != "none" {
		t.Fatalf("GetBalance() meta err = %q, want %q", gotMeta.Err, "none")
	}

	if gotMeta.StartedAt != 123 || gotMeta.EndedAt != 456 {
		t.Fatalf("GetBalance() meta times = %d/%d, want 123/456", gotMeta.StartedAt, gotMeta.EndedAt)
	}
}

func TestInMemoryStore_SaveResults_And_ListIssues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewInMemoryStore()
	meta := entity.UploadMeta{
		ID:     "upload-3",
		Status: entity.UploadStatusProcessing,
	}

	if err := store.CreateUpload(ctx, meta); err != nil {
		t.Fatalf("CreateUpload() err = %v", err)
	}

	issues := []entity.Transaction{
		{Timestamp: 1, Counterparty: "A", Type: entity.TxTypeCredit, Amount: 100, Status: entity.TxStatusSuccess, Description: "ok"},
		{Timestamp: 2, Counterparty: "B", Type: entity.TxTypeDebit, Amount: 50, Status: entity.TxStatusFailed, Description: "fail-1"},
		{Timestamp: 3, Counterparty: "C", Type: entity.TxTypeCredit, Amount: 70, Status: entity.TxStatusFailed, Description: "fail-2"},
		{Timestamp: 4, Counterparty: "D", Type: entity.TxTypeDebit, Amount: 30, Status: entity.TxStatusPending, Description: "pending"},
	}

	if err := store.SaveResults(ctx, meta.ID, 500, issues, 4, 3, 1); err != nil {
		t.Fatalf("SaveResults() err = %v", err)
	}

	balance, gotMeta, err := store.GetBalance(ctx, meta.ID)
	if err != nil {
		t.Fatalf("GetBalance() err = %v", err)
	}
	if balance != 500 {
		t.Fatalf("GetBalance() balance = %d, want 500", balance)
	}
	if gotMeta.TotalLines != 4 || gotMeta.ParsedOK != 3 || gotMeta.ParseErr != 1 {
		t.Fatalf("GetBalance() meta stats = %d/%d/%d, want 4/3/1", gotMeta.TotalLines, gotMeta.ParsedOK, gotMeta.ParseErr)
	}

	filterFailed := usecase.IssueFilter{Statuses: []entity.TxStatus{entity.TxStatusFailed}}
	page1, total, metaOut, err := store.ListIssues(ctx, meta.ID, filterFailed, 1, 1)
	if err != nil {
		t.Fatalf("ListIssues() err = %v", err)
	}
	if total != 2 {
		t.Fatalf("ListIssues() total = %d, want 2", total)
	}
	if len(page1) != 1 {
		t.Fatalf("ListIssues() page1 len = %d, want 1", len(page1))
	}
	if !reflect.DeepEqual(page1[0], issues[1]) {
		t.Fatalf("ListIssues() page1 item = %+v, want %+v", page1[0], issues[1])
	}
	if metaOut.TotalLines != 4 || metaOut.ParsedOK != 3 || metaOut.ParseErr != 1 {
		t.Fatalf("ListIssues() meta stats = %d/%d/%d, want 4/3/1", metaOut.TotalLines, metaOut.ParsedOK, metaOut.ParseErr)
	}

	page2, total, _, err := store.ListIssues(ctx, meta.ID, filterFailed, 2, 1)
	if err != nil {
		t.Fatalf("ListIssues() page2 err = %v", err)
	}
	if total != 2 {
		t.Fatalf("ListIssues() page2 total = %d, want 2", total)
	}
	if len(page2) != 1 {
		t.Fatalf("ListIssues() page2 len = %d, want 1", len(page2))
	}
	if !reflect.DeepEqual(page2[0], issues[2]) {
		t.Fatalf("ListIssues() page2 item = %+v, want %+v", page2[0], issues[2])
	}

	filterFailedCredit := usecase.IssueFilter{
		Statuses: []entity.TxStatus{entity.TxStatusFailed},
		Types:    []entity.TxType{entity.TxTypeCredit},
	}
	matches, total, _, err := store.ListIssues(ctx, meta.ID, filterFailedCredit, 1, 10)
	if err != nil {
		t.Fatalf("ListIssues() filtered err = %v", err)
	}
	if total != 1 {
		t.Fatalf("ListIssues() filtered total = %d, want 1", total)
	}
	if len(matches) != 1 {
		t.Fatalf("ListIssues() filtered len = %d, want 1", len(matches))
	}
	if !reflect.DeepEqual(matches[0], issues[2]) {
		t.Fatalf("ListIssues() filtered item = %+v, want %+v", matches[0], issues[2])
	}
}

func TestInMemoryStore_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewInMemoryStore()

	t.Run("GetBalance", func(t *testing.T) {
		_, _, err := store.GetBalance(ctx, "missing")
		if !errors.Is(err, pkgerror.ErrNotFound) {
			t.Fatalf("GetBalance() err = %v, want ErrNotFound", err)
		}
	})

	t.Run("UpdateMeta", func(t *testing.T) {
		err := store.UpdateMeta(ctx, "missing", func(*entity.UploadMeta) {})
		if !errors.Is(err, pkgerror.ErrNotFound) {
			t.Fatalf("UpdateMeta() err = %v, want ErrNotFound", err)
		}
	})

	t.Run("SaveResults", func(t *testing.T) {
		err := store.SaveResults(ctx, "missing", 0, nil, 0, 0, 0)
		if !errors.Is(err, pkgerror.ErrNotFound) {
			t.Fatalf("SaveResults() err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListIssues", func(t *testing.T) {
		_, _, _, err := store.ListIssues(ctx, "missing", usecase.IssueFilter{}, 1, 10)
		if !errors.Is(err, pkgerror.ErrNotFound) {
			t.Fatalf("ListIssues() err = %v, want ErrNotFound", err)
		}
	})
}
