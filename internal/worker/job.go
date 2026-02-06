package worker

import (
	"context"
	"time"

	"mysql-exporter/internal/exporter"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusPending    JobStatus = "PENDING"
	StatusProcessing JobStatus = "PROCESSING"
	StatusCompleted  JobStatus = "COMPLETED"
	StatusFailed     JobStatus = "FAILED"
)

// ExportJob represents a single unit of work for the export service.
type ExportJob struct {
	// ID is the unique UUID v4 for the job.
	ID string
	// Query is the SQL SELECT statement execution.
	Query string
	// Email is the recipient address for notifications.
	Email string
	// Timestamps for job lifecycle tracking.
	Submitted time.Time
	Started   time.Time
	Finished  time.Time
	// Status tracks the current state (PENDING, PROCESSING, COMPLETED, FAILED).
	Status JobStatus
	// Error holds any error encountered during processing.
	Error error
	// Stats contains metrics like rows processed and duration.
	Stats *exporter.ExportResult
	// S3Key is the path where the file is stored in S3/Local storage.
	S3Key string
	// Format is the requested output format (csv, json, excel).
	Format string

	// Context manages the lifecycle/cancellation of the job.
	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewExportJob(query, email, format string, timeout time.Duration) *ExportJob {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	if format == "" {
		format = "csv"
	}
	return &ExportJob{
		ID:        uuid.New().String(),
		Query:     query,
		Email:     email,
		Format:    format,
		Submitted: time.Now(),
		Status:    StatusPending,
		Ctx:       ctx,
		Cancel:    cancel,
	}
}
