package worker

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"mysql-exporter/internal/email"
	"mysql-exporter/internal/exporter"
	"mysql-exporter/internal/storage"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Pool manages concurrent export jobs and limits database load.
// It implements a worker pool pattern with a separate semaphore for DB connections,
// allowing for fine-grained control over resource usage.
type Pool struct {
	// jobQueue allows for buffering incoming requests before workers pick them up.
	jobQueue chan *ExportJob
	workers  int
	// dbSem restricts the number of concurrent queries to the database.
	dbSem *semaphore.Weighted
	wg    sync.WaitGroup
	quit  chan struct{}

	db         *sql.DB
	storage    storage.Provider
	emailer    email.Sender
	useGzip    bool
	attachFile bool
}

// NewPool initializes a worker pool with the specified configuration.
// It does not start the workers; call Start() to begin processing.
func NewPool(workers int, maxDBConcurrency int64, db *sql.DB, store storage.Provider, emailer email.Sender, useGzip, attachFile bool) *Pool {
	return &Pool{
		jobQueue:   make(chan *ExportJob, 100), // Bounded buffer to prevent infinite memory growth
		workers:    workers,
		dbSem:      semaphore.NewWeighted(maxDBConcurrency),
		quit:       make(chan struct{}),
		db:         db,
		storage:    store,
		emailer:    emailer,
		useGzip:    useGzip,
		attachFile: attachFile,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.workerLoop(i)
	}
	slog.Info("Worker pool started", "workers", p.workers)
}

func (p *Pool) Submit(job *ExportJob) bool {
	select {
	case p.jobQueue <- job:
		return true
	case <-p.quit:
		return false
	default:
		// Queue full
		return false
	}
}

// Stop initiates graceful shutdown
func (p *Pool) Stop() {
	close(p.quit)
	p.wg.Wait()
	slog.Info("Worker pool stopped")
}

func (p *Pool) workerLoop(id int) {
	defer p.wg.Done()
	slog.Debug("Worker started", "worker_id", id)

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				return
			}
			p.processJob(id, job)
		case <-p.quit:
			return
		}
	}
}

func (p *Pool) processJob(workerID int, job *ExportJob) {
	slog.Info("Processing job", "worker_id", workerID, "job_id", job.ID)

	job.Started = time.Now()
	job.Status = StatusProcessing
	waitTime := job.Started.Sub(job.Submitted)

	// 1. Acquire DB Semaphore
	if err := p.dbSem.Acquire(job.Ctx, 1); err != nil {
		p.failJob(job, fmt.Errorf("failed to acquire db connection: %w", err))
		return
	}

	err := p.executeExport(job)
	p.dbSem.Release(1)

	if err != nil {
		p.failJob(job, err)
		return
	}

	job.Status = StatusCompleted
	job.Finished = time.Now()
	totalDuration := job.Finished.Sub(job.Started)

	slog.Info("Job completed", "job_id", job.ID, "rows", job.Stats.RowsProcessed)

	// Build detailed report
	statsMsg := fmt.Sprintf(
		"Job Summary:\n"+
			"----------------\n"+
			"Job ID: %s\n"+
			"Rows Processed: %d\n"+
			"Submitted: %s\n"+
			"Started: %s (Wait: %v)\n"+
			"Finished: %s\n"+
			"Total Duration: %v\n"+
			"Query Execution: %v\n",
		job.ID,
		job.Stats.RowsProcessed,
		job.Submitted.Format("2006-01-02 03:04:05 PM"),
		job.Started.Format("2006-01-02 03:04:05 PM"), waitTime,
		job.Finished.Format("2006-01-02 03:04:05 PM"),
		totalDuration,
		job.Stats.Duration,
	)

	const MaxAttachmentSize = 25 * 1024 * 1024 // 25MB

	if p.attachFile {
		fileContent, err := func() ([]byte, error) {
			reader, err := p.storage.OpenFile(job.Ctx, job.S3Key)
			if err != nil {
				return nil, err
			}
			defer reader.Close()

			limitedReader := io.LimitReader(reader, MaxAttachmentSize+1)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				return nil, err
			}

			if len(content) > MaxAttachmentSize {
				return nil, fmt.Errorf("file exceeds max attachment size (%d bytes)", MaxAttachmentSize)
			}
			return content, nil
		}()

		if err != nil {
			slog.Warn("Skipping attachment (too large or error)", "key", job.S3Key, "error", err)
			downloadURL := p.storage.GetDownloadURL(job.S3Key)
			statsMsg += fmt.Sprintf("\nAttachment skipped: %v\nDownload Link: %s", err, downloadURL)
			p.emailer.SendDownloadLink(job.Email, downloadURL, statsMsg)
		} else {
			p.emailer.SendWithAttachment(job.Email, job.S3Key, fileContent, statsMsg)
		}

	} else {
		downloadURL := p.storage.GetDownloadURL(job.S3Key)
		p.emailer.SendDownloadLink(job.Email, downloadURL, statsMsg)
	}
}

func (p *Pool) executeExport(job *ExportJob) error {
	// Setup Pipeline
	ext := job.Format
	if ext == "" {
		ext = "csv"
	}
	if ext == "excel" {
		ext = "xlsx"
	}

	if p.useGzip {
		job.S3Key = fmt.Sprintf("exports/%s.%s.gz", job.ID, ext)
	} else {
		job.S3Key = fmt.Sprintf("exports/%s.%s", job.ID, ext)
	}

	// Start Storage Upload in background (it reads from pipe)
	storageWriter, errChan := p.storage.StreamToFile(job.Ctx, job.S3Key)

	// Prepare Output Writer (maybe wrapped in Gzip)
	var finalWriter io.WriteCloser
	if p.useGzip {
		finalWriter = gzip.NewWriter(storageWriter)
	} else {
		finalWriter = storageWriter
	}

	// Choose Encoder
	var encoder exporter.RowEncoder
	switch job.Format {
	case "json":
		encoder = exporter.NewJSONEncoder(finalWriter)
	case "excel":
		encoder = exporter.NewExcelEncoder(finalWriter)
	case "pdf":
		encoder = exporter.NewPDFEncoder(finalWriter)
	default:
		encoder = exporter.NewCSVEncoder(finalWriter)
	}

	// Prepare MySQL Streamer
	mysqlStreamer := exporter.NewMySQLStreamer(p.db)

	// Run Export (DB -> Encoder -> [Gzip?] -> Pipe -> Storage)
	stats, exportErr := mysqlStreamer.StreamQuery(job.Ctx, job.Query, encoder)

	// Close Encoder (some formats need to finish writing/flushing)
	encoderCloseErr := encoder.Close()

	// Close Writers
	// If Gzip, close it first to flush footer
	var outputCloseErr error
	if gw, ok := finalWriter.(*gzip.Writer); ok {
		outputCloseErr = gw.Close()
	}

	// Then close the underlying storage writer (the pipe)
	storageCloseErr := storageWriter.Close()

	// Wait for upload result
	uploadErr := <-errChan

	if exportErr != nil {
		return fmt.Errorf("export failed: %w", exportErr)
	}
	if encoderCloseErr != nil {
		return fmt.Errorf("encoder close failed: %w", encoderCloseErr)
	}
	if outputCloseErr != nil {
		return fmt.Errorf("gzip close failed: %w", outputCloseErr)
	}
	if storageCloseErr != nil {
		return fmt.Errorf("storage close failed: %w", storageCloseErr)
	}
	if uploadErr != nil {
		return fmt.Errorf("upload failed: %w", uploadErr)
	}

	job.Stats = stats
	return nil
}

func (p *Pool) failJob(job *ExportJob, err error) {
	job.Status = StatusFailed
	job.Error = err
	job.Finished = time.Now()
	slog.Error("Job failed", "job_id", job.ID, "error", err)
}
