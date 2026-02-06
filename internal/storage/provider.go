package storage

import (
	"context"
	"io"
)

// Provider defines the interface for storing exported data.
type Provider interface {
	// StreamToFile returns a WriteCloser. Data written to it is streamed to the storage destination.
	// The key is the relative path/filename for the object.
	// The returned channel receives a single error (or nil) when the storage operation completes.
	StreamToFile(ctx context.Context, key string) (io.WriteCloser, <-chan error)

	// OpenFile opens the stored file for reading.
	OpenFile(ctx context.Context, key string) (io.ReadCloser, error)

	// GetDownloadURL returns a viewable/downloadable URL for the stored item.
	GetDownloadURL(key string) string
}
