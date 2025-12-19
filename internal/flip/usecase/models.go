package usecase

import (
	"slices"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
)

type UploadResult struct {
	UploadID string
}

type BalanceResult struct {
	UploadID string
	Status   entity.UploadStatus
	Balance  int64
}

type IssuesResult struct {
	UploadID     string
	Status       entity.UploadStatus
	Transactions []entity.Transaction
	Page         int
	PageSize     int
	Total        int
}

type IssueFilter struct {
	Statuses []entity.TxStatus
	Types    []entity.TxType
}

func (f IssueFilter) Matches(tx entity.Transaction) bool {
	if len(f.Statuses) > 0 {
		ok := slices.Contains(f.Statuses, tx.Status)
		if !ok {
			return false
		}
	}

	if len(f.Types) > 0 {
		ok := false
		for _, typ := range f.Types {
			if tx.Type == typ {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	return true
}
