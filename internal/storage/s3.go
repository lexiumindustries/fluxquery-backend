package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Provider struct {
	client *s3.Client
	bucket string
}

func NewS3Provider(client *s3.Client, bucket string) *S3Provider {
	return &S3Provider{
		client: client,
		bucket: bucket,
	}
}

func (p *S3Provider) StreamToFile(ctx context.Context, key string) (io.WriteCloser, <-chan error) {
	// NOTE: We do NOT do gzip compression here anymore if we want the provider to be generic.
	// OR we assume the provider handles compression?
	// The previous implementation had the GzipWriter inside the s3_stream.go.
	// If we move to Local, we likely want Gzip there too.
	// So the "StreamToFile" should probably receive already-compressed bytes OR we enforce compression in the provider.
	// For now, I will keep Gzip inside the provider to match previous behavior, so Local files are also gzipped.

	reader, writer := io.Pipe()
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		uploader := manager.NewUploader(p.client, func(u *manager.Uploader) {
			u.PartSize = 10 * 1024 * 1024 // 10MB chunks
			u.Concurrency = 5
		})

		slog.Info("Starting S3 upload", "key", key)
		_, err := uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(p.bucket),
			Key:    aws.String(key),
			Body:   reader,
		})

		_ = reader.Close()

		if err != nil {
			slog.Error("S3 Upload failed", "error", err)
			errChan <- fmt.Errorf("s3 upload failed: %w", err)
		} else {
			slog.Info("S3 Upload finished successfully", "key", key)
			errChan <- nil
		}
	}()

	return writer, errChan
}

func (p *S3Provider) OpenFile(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (p *S3Provider) GetDownloadURL(key string) string {
	return fmt.Sprintf("s3://%s/%s", p.bucket, key)
}
