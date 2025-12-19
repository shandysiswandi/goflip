package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shandysiswandi/goflip/internal/flip/entity"
)

func parseCSV(ctx context.Context, r io.Reader, onTx func(tx entity.Transaction)) (int64, int64, int64, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = true
	reader.FieldsPerRecord = -1

	var totalLines int64
	var parsedOK int64
	var parseErr int64

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			parseErr++
			slog.WarnContext(ctx, "failed to read csv line", "error", err)
			return totalLines, parsedOK, parseErr, err
		}

		totalLines++
		tx, err := parseRecord(record)
		if err != nil {
			parseErr++
			slog.WarnContext(ctx, "failed to parse csv record", "error", err)
			continue
		}

		parsedOK++
		onTx(tx)
	}

	return totalLines, parsedOK, parseErr, nil
}

func parseRecord(record []string) (entity.Transaction, error) {
	if len(record) != 6 {
		return entity.Transaction{}, fmt.Errorf("expected 6 fields, got %d", len(record))
	}

	for i := range record {
		record[i] = strings.TrimSpace(record[i])
	}

	timestamp, err := strconv.ParseInt(record[0], 10, 64)
	if err != nil {
		return entity.Transaction{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	txType, err := parseTxType(record[2])
	if err != nil {
		return entity.Transaction{}, err
	}

	amount, err := strconv.ParseInt(record[3], 10, 64)
	if err != nil {
		return entity.Transaction{}, fmt.Errorf("invalid amount: %w", err)
	}

	status, err := parseTxStatus(record[4])
	if err != nil {
		return entity.Transaction{}, err
	}

	return entity.Transaction{
		Timestamp:    timestamp,
		Counterparty: record[1],
		Type:         txType,
		Amount:       amount,
		Status:       status,
		Description:  record[5],
	}, nil
}

func parseTxType(value string) (entity.TxType, error) {
	switch strings.ToUpper(value) {
	case string(entity.TxTypeCredit):
		return entity.TxTypeCredit, nil
	case string(entity.TxTypeDebit):
		return entity.TxTypeDebit, nil
	default:
		return "", fmt.Errorf("invalid tx type: %s", value)
	}
}

func parseTxStatus(value string) (entity.TxStatus, error) {
	switch strings.ToUpper(value) {
	case string(entity.TxStatusSuccess):
		return entity.TxStatusSuccess, nil
	case string(entity.TxStatusFailed):
		return entity.TxStatusFailed, nil
	case string(entity.TxStatusPending):
		return entity.TxStatusPending, nil
	default:
		return "", fmt.Errorf("invalid tx status: %s", value)
	}
}
