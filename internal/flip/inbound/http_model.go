package inbound

import (
	"net/http"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
)

type Transaction struct {
	Timestamp    int64           `json:"timestamp"`
	Counterparty string          `json:"counterparty"`
	Type         entity.TxType   `json:"type"`
	Amount       int64           `json:"amount"`
	Status       entity.TxStatus `json:"status"`
	Description  string          `json:"description"`
}

type UploadResponse struct {
	UploadID string `json:"upload_id"`
}

func (UploadResponse) StatusCode() int {
	return http.StatusAccepted
}

func (UploadResponse) Message() string {
	return "upload accepted"
}

type BalanceResponse struct {
	UploadID string              `json:"upload_id"`
	Status   entity.UploadStatus `json:"status"`
	Balance  int64               `json:"balance"`
}

type TransactionIssuesResponse struct {
	UploadID     string              `json:"upload_id"`
	Status       entity.UploadStatus `json:"status"`
	Transactions []Transaction       `json:"transactions"`
	page         int
	pageSize     int
	total        int
}

func (r TransactionIssuesResponse) Meta() map[string]any {
	return map[string]any{
		"page":      r.page,
		"page_size": r.pageSize,
		"total":     r.total,
	}
}
