package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgerror"
	"github.com/shandysiswandi/goflip/internal/pkg/pkguid"
)

type Store interface {
	CreateUpload(ctx context.Context, meta entity.UploadMeta) error
	UpdateMeta(ctx context.Context, uploadID string, fn func(meta *entity.UploadMeta)) error
	SaveResults(ctx context.Context, uploadID string, balance int64, issues []entity.Transaction, totalLines, parsedOK, parseErr int64) error
	GetBalance(ctx context.Context, uploadID string) (int64, entity.UploadMeta, error)
	ListIssues(ctx context.Context, uploadID string, filter IssueFilter, page, pageSize int) ([]entity.Transaction, int, entity.UploadMeta, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event entity.FailedTxEvent) error
}

type Runner interface {
	Go(ctx context.Context, f func(ctx context.Context) error)
}

type Clock interface {
	Now() time.Time
}

type Dependency struct {
	Store   Store
	Events  EventPublisher
	Runner  Runner
	Clock   Clock
	ID      pkguid.StringID
	RootCtx context.Context
}

type Usecase struct {
	store   Store
	events  EventPublisher
	runner  Runner
	clock   Clock
	id      pkguid.StringID
	rootCtx context.Context
}

func New(dep Dependency) *Usecase {
	root := dep.RootCtx
	if root == nil {
		root = context.Background()
	}

	clock := dep.Clock
	if clock == nil {
		clock = realClock{}
	}

	return &Usecase{
		store:   dep.Store,
		events:  dep.Events,
		runner:  dep.Runner,
		clock:   clock,
		id:      dep.ID,
		rootCtx: root,
	}
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (u *Usecase) Upload(ctx context.Context, r io.Reader) (UploadResult, error) {
	if u.store == nil || u.id == nil || u.runner == nil {
		return UploadResult{}, pkgerror.NewServer(errors.New("missing dependency"))
	}

	uploadID := u.id.Generate()
	if err := u.store.CreateUpload(ctx, entity.UploadMeta{
		ID:     uploadID,
		Status: entity.UploadStatusQueued,
	}); err != nil {
		return UploadResult{}, normalizeErr(err)
	}

	u.runner.Go(u.rootCtx, func(ctx context.Context) error {
		if err := u.processUpload(ctx, uploadID, r); err != nil {
			slog.ErrorContext(ctx, "upload processing failed", "upload_id", uploadID, "error", err)
			return err
		}
		return nil
	})

	return UploadResult{UploadID: uploadID}, nil
}

func (u *Usecase) Balance(ctx context.Context, uploadID string) (BalanceResult, error) {
	if uploadID == "" {
		return BalanceResult{}, pkgerror.NewInvalidInput(errors.New("upload_id is required"))
	}

	balance, meta, err := u.store.GetBalance(ctx, uploadID)
	if err != nil {
		return BalanceResult{}, mapStoreErr(err)
	}

	return BalanceResult{
		UploadID: uploadID,
		Status:   meta.Status,
		Balance:  balance,
	}, nil
}

func (u *Usecase) Issues(ctx context.Context, uploadID string, filter IssueFilter, page, pageSize int) (IssuesResult, error) {
	if uploadID == "" {
		return IssuesResult{}, pkgerror.NewInvalidInput(errors.New("upload_id is required"))
	}

	if page < 1 || pageSize < 1 {
		return IssuesResult{}, pkgerror.NewInvalidInput(errors.New("invalid pagination"))
	}

	issues, total, meta, err := u.store.ListIssues(ctx, uploadID, filter, page, pageSize)
	if err != nil {
		return IssuesResult{}, mapStoreErr(err)
	}

	return IssuesResult{
		UploadID:     uploadID,
		Status:       meta.Status,
		Transactions: issues,
		Page:         page,
		PageSize:     pageSize,
		Total:        total,
	}, nil
}

func (u *Usecase) processUpload(ctx context.Context, uploadID string, r io.Reader) error {
	startedAt := u.clock.Now().Unix()
	if err := u.store.UpdateMeta(ctx, uploadID, func(meta *entity.UploadMeta) {
		meta.Status = entity.UploadStatusProcessing
		meta.StartedAt = startedAt
	}); err != nil {
		return err
	}

	var balance int64
	var issues []entity.Transaction

	totalLines, parsedOK, parseErr, err := parseCSV(ctx, r, func(tx entity.Transaction) {
		if tx.Status == entity.TxStatusSuccess {
			switch tx.Type {
			case entity.TxTypeCredit:
				balance += tx.Amount
			case entity.TxTypeDebit:
				balance -= tx.Amount
			}
			return
		}

		issues = append(issues, tx)
		if tx.Status == entity.TxStatusFailed && u.events != nil {
			event := entity.FailedTxEvent{
				EventID:  u.id.Generate(),
				UploadID: uploadID,
				Tx:       tx,
			}
			if pubErr := u.events.Publish(ctx, event); pubErr != nil {
				slog.WarnContext(ctx, "failed to publish event", "upload_id", uploadID, "event_id", event.EventID, "error", pubErr)
			}
		}
	})

	endedAt := u.clock.Now().Unix()
	status := entity.UploadStatusDone
	errMsg := ""
	if err != nil {
		status = entity.UploadStatusFailed
		errMsg = err.Error()
	}

	if saveErr := u.store.SaveResults(ctx, uploadID, balance, issues, totalLines, parsedOK, parseErr); saveErr != nil {
		return saveErr
	}

	if metaErr := u.store.UpdateMeta(ctx, uploadID, func(meta *entity.UploadMeta) {
		meta.Status = status
		meta.Err = errMsg
		meta.EndedAt = endedAt
		meta.TotalLines = totalLines
		meta.ParsedOK = parsedOK
		meta.ParseErr = parseErr
	}); metaErr != nil {
		return metaErr
	}

	return err
}

func mapStoreErr(err error) error {
	if errors.Is(err, pkgerror.ErrNotFound) {
		return pkgerror.NewBusiness("upload not found", pkgerror.CodeNotFound)
	}
	return normalizeErr(err)
}

func normalizeErr(err error) error {
	var perr *pkgerror.Error
	if errors.As(err, &perr) {
		return perr
	}
	return pkgerror.NewServer(err)
}
