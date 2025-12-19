package entity

type UploadMeta struct {
	ID        string
	Status    UploadStatus
	Err       string
	StartedAt int64
	EndedAt   int64

	// Stats help observability without storing everything
	TotalLines int64
	ParsedOK   int64
	ParseErr   int64
}
