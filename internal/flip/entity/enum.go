package entity

type TxType string

const (
	TxTypeCredit TxType = "CREDIT"
	TxTypeDebit  TxType = "DEBIT"
)

type TxStatus string

const (
	TxStatusSuccess TxStatus = "SUCCESS"
	TxStatusFailed  TxStatus = "FAILED"
	TxStatusPending TxStatus = "PENDING"
)

type UploadStatus string

const (
	UploadStatusQueued     UploadStatus = "QUEUED"
	UploadStatusProcessing UploadStatus = "PROCESSING"
	UploadStatusDone       UploadStatus = "DONE"
	UploadStatusFailed     UploadStatus = "FAILED"
)
