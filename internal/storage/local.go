package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type LocalProvider struct {
	basePath string
}

func NewLocalProvider(basePath string) *LocalProvider {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		slog.Error("Failed to ensure local storage directory exists", "path", basePath, "error", err)
	}
	return &LocalProvider{
		basePath: basePath,
	}
}

func (p *LocalProvider) StreamToFile(ctx context.Context, key string) (io.WriteCloser, <-chan error) {
	errChan := make(chan error, 1)

	// Ensure subdirectories exist if key contains them
	fullPath := filepath.Join(p.basePath, key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		errChan <- fmt.Errorf("failed to create directory %s: %w", dir, err)
		close(errChan)
		return nil, errChan
	}

	f, err := os.Create(fullPath)
	if err != nil {
		errChan <- fmt.Errorf("failed to create file %s: %w", fullPath, err)
		close(errChan)
		return nil, errChan
	}

	// We wrap the file to handle closing the error channel on Close()
	return &localWriter{
		f:       f,
		errChan: errChan,
		path:    fullPath,
	}, errChan
}

func (p *LocalProvider) OpenFile(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(p.basePath, key)
	return os.Open(fullPath)
}

func (p *LocalProvider) GetDownloadURL(key string) string {
	// For local, we just return the absolute path for now,
	// or a file:// URL.
	fullPath := filepath.Join(p.basePath, key)
	abs, _ := filepath.Abs(fullPath)
	return fmt.Sprintf("file://%s", abs)
}

type localWriter struct {
	f       *os.File
	errChan chan error
	path    string
}

func (w *localWriter) Write(p []byte) (n int, err error) {
	return w.f.Write(p)
}

func (w *localWriter) Close() error {
	err := w.f.Close()
	if err != nil {
		w.errChan <- err
	} else {
		slog.Info("Local file write completed", "path", w.path)
		w.errChan <- nil
	}
	close(w.errChan)
	return err
}
