package entity

type FailedTxEvent struct {
	EventID  string
	UploadID string
	Tx       Transaction
}
