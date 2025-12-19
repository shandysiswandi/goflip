package inbound

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
	"github.com/shandysiswandi/goflip/internal/flip/usecase"
	"github.com/shandysiswandi/goflip/internal/pkg/pkgerror"
)

type HTTPEndpoint struct {
	uc uc
}

func (h *HTTPEndpoint) Statements(ctx context.Context, r *http.Request) (any, error) {
	reader, cleanup, err := extractCSVReader(r)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	pr, pw := io.Pipe()
	result, err := h.uc.Upload(ctx, pr)
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return nil, err
	}

	if err := streamToPipe(reader, pw); err != nil {
		return nil, pkgerror.NewServer(err)
	}

	return UploadResponse{UploadID: result.UploadID}, nil
}

func (h *HTTPEndpoint) Balance(ctx context.Context, r *http.Request) (any, error) {
	uploadID := strings.TrimSpace(r.URL.Query().Get("upload_id"))
	if uploadID == "" {
		return nil, pkgerror.NewInvalidInput(errors.New("upload_id is required"))
	}

	result, err := h.uc.Balance(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	return BalanceResponse{
		UploadID: result.UploadID,
		Status:   result.Status,
		Balance:  result.Balance,
	}, nil
}

func (h *HTTPEndpoint) TransactionIssues(ctx context.Context, r *http.Request) (any, error) {
	query := r.URL.Query()
	uploadID := strings.TrimSpace(query.Get("upload_id"))
	if uploadID == "" {
		return nil, pkgerror.NewInvalidInput(errors.New("upload_id is required"))
	}

	page, pageSize, err := parsePagination(query.Get("page"), query.Get("page_size"))
	if err != nil {
		return nil, err
	}

	filter, err := parseIssueFilter(query.Get("status"), query.Get("type"))
	if err != nil {
		return nil, err
	}

	result, err := h.uc.Issues(ctx, uploadID, filter, page, pageSize)
	if err != nil {
		return nil, err
	}

	transactions := make([]Transaction, 0, len(result.Transactions))
	for _, tx := range result.Transactions {
		transactions = append(transactions, toHTTPTransaction(tx))
	}

	return TransactionIssuesResponse{
		UploadID:     result.UploadID,
		Status:       result.Status,
		Transactions: transactions,
		page:         result.Page,
		pageSize:     result.PageSize,
		total:        result.Total,
	}, nil
}

func parsePagination(pageRaw, sizeRaw string) (int, int, error) {
	page := 1
	pageSize := 10

	if pageRaw != "" {
		value, err := strconv.Atoi(pageRaw)
		if err != nil || value < 1 {
			return 0, 0, pkgerror.NewInvalidInput(errors.New("invalid page"))
		}
		page = value
	}

	if sizeRaw != "" {
		value, err := strconv.Atoi(sizeRaw)
		if err != nil || value < 1 {
			return 0, 0, pkgerror.NewInvalidInput(errors.New("invalid page_size"))
		}
		if value > 100 {
			value = 100
		}
		pageSize = value
	}

	return page, pageSize, nil
}

func parseIssueFilter(statusRaw, typeRaw string) (usecase.IssueFilter, error) {
	filter := usecase.IssueFilter{}

	if statusRaw != "" {
		statuses := strings.Split(statusRaw, ",")
		for _, value := range statuses {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			status, err := parseStatus(value)
			if err != nil {
				return filter, err
			}
			filter.Statuses = append(filter.Statuses, status)
		}
	}

	if typeRaw != "" {
		types := strings.Split(typeRaw, ",")
		for _, value := range types {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			typ, err := parseType(value)
			if err != nil {
				return filter, err
			}
			filter.Types = append(filter.Types, typ)
		}
	}

	if len(filter.Statuses) == 0 {
		filter.Statuses = []entity.TxStatus{entity.TxStatusFailed, entity.TxStatusPending}
	}

	return filter, nil
}

func parseStatus(value string) (entity.TxStatus, error) {
	switch strings.ToUpper(value) {
	case string(entity.TxStatusFailed):
		return entity.TxStatusFailed, nil
	case string(entity.TxStatusPending):
		return entity.TxStatusPending, nil
	default:
		return "", pkgerror.NewInvalidInput(errors.New("invalid status filter"))
	}
}

func parseType(value string) (entity.TxType, error) {
	switch strings.ToUpper(value) {
	case string(entity.TxTypeCredit):
		return entity.TxTypeCredit, nil
	case string(entity.TxTypeDebit):
		return entity.TxTypeDebit, nil
	default:
		return "", pkgerror.NewInvalidInput(errors.New("invalid type filter"))
	}
}

func toHTTPTransaction(tx entity.Transaction) Transaction {
	return Transaction{
		Timestamp:    tx.Timestamp,
		Counterparty: tx.Counterparty,
		Type:         tx.Type,
		Amount:       tx.Amount,
		Status:       tx.Status,
		Description:  tx.Description,
	}
}

func extractCSVReader(r *http.Request) (io.ReadCloser, func(), error) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
			return extractMultipartFile(r)
		}
	}

	if r.Body == nil {
		return nil, func() {}, pkgerror.NewInvalidInput(errors.New("empty request body"))
	}

	return r.Body, func() {}, nil
}

func extractMultipartFile(r *http.Request) (io.ReadCloser, func(), error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, func() {}, pkgerror.NewInvalidFormat()
	}

	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, func() {}, pkgerror.NewInvalidInput(errors.New("file part is required"))
			}
			return nil, func() {}, pkgerror.NewInvalidFormat()
		}

		if part.FormName() == "file" {
			return part, func() { _ = part.Close() }, nil
		}
		_ = part.Close()
	}
}

func streamToPipe(src io.Reader, dst *io.PipeWriter) error {
	defer func() {
		_ = dst.Close()
	}()

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.CloseWithError(err)
		return err
	}

	return nil
}
