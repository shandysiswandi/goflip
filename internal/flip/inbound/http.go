package inbound

import (
	"context"
	"io"

	"github.com/shandysiswandi/goflip/internal/flip/usecase"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgrouter"
)

type uc interface {
	Upload(ctx context.Context, r io.Reader) (usecase.UploadResult, error)
	Balance(ctx context.Context, uploadID string) (usecase.BalanceResult, error)
	Issues(ctx context.Context, uploadID string, filter usecase.IssueFilter, page, pageSize int) (usecase.IssuesResult, error)
}

func RegisterHTTPEndpoint(r *pkgrouter.Router, uc uc) {
	end := &HTTPEndpoint{uc: uc}

	r.POST("/statements", end.Statements)

	r.GET("/balance", end.Balance)                       // ?upload_id=
	r.GET("/transactions/issues", end.TransactionIssues) // ?upload_id=
}
