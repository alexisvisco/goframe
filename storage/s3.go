package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/http/httpx"
	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nrednav/cuid2"
)

type (
	S3Storage struct {
		config     configuration.Storage
		repository contracts.StorageRepository
		client     *minio.Client
	}
)

var _ contracts.Storage = (*S3Storage)(nil)

func NewS3Storage(
	cfg configuration.Storage,
	repository contracts.StorageRepository,
) (*S3Storage, error) {
	// Initialize MinIO client
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.Secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client: %w", err)
	}

	return &S3Storage{
		config:     cfg,
		client:     client,
		repository: repository,
	}, nil
}

func (s *S3Storage) UploadAttachment(ctx context.Context, opts coretypes.UploadAttachmentOptions) (*coretypes.Attachment, error) {
	// WriteTo a unique ID for the attachment if not replacing existing
	id := cuid2.Generate()
	if opts.CurrentAttachmentID != nil {
		id = *opts.CurrentAttachmentID
	}

	// GenerateHandler temporary file
	tmpFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	// Calculate sha256 and size while writing to temp file
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmpFile, hash), opts.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	// Rewind temp file for content type detection
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	buffer := make([]byte, 3072)
	_, err = tmpFile.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read temp file: %w", err)
	}
	contentType := mimetype.Detect(buffer)

	// Rewind file again for upload
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	// Construct the object key
	key := path.Join("attachments", fmt.Sprintf("%s-%s", id, str.Slugify(opts.Filename)))

	var attachment *coretypes.Attachment

	_, err = s.client.PutObject(
		ctx,
		s.config.Bucket,
		key,
		tmpFile,
		size,
		minio.PutObjectOptions{
			ContentType: contentType.String(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}

	attachment = &coretypes.Attachment{
		ID:          id,
		Filename:    opts.Filename,
		ContentType: contentType.String(),
		ByteSize:    size,
		Key:         key,
		Checksum:    hex.EncodeToString(hash.Sum(nil)),
		CreatedAt:   time.Now(),
	}
	err = s.repository.SaveAttachment(ctx, attachment)
	if err != nil {
		slog.Warn("attachment saved to s3 but failed to save record in database", "attachment_id", id, "error", err)
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	return attachment, err
}

func (s *S3Storage) DeleteAttachment(ctx context.Context, id string) error {
	attachment, err := s.repository.GetAttachment(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch attachment: %w", err)
	}

	if attachment == nil {
		return nil
	}

	err = s.repository.DeleteAttachment(ctx, id, func() error {
		err = s.client.RemoveObject(ctx, s.config.Bucket, attachment.Key, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete from storage: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

func (s *S3Storage) AttachmentHandler(pathValueField string) http.HandlerFunc {
	return httpx.Wrap(func(r *http.Request) (httpx.Response, error) {
		// Extract attachment ID from path parameter
		id := r.PathValue(pathValueField)
		if id == "" {
			return httpx.JSON.BadRequest("Attachment ID is required"), nil
		}

		attachment, err := s.repository.GetAttachment(r.Context(), id)
		if err != nil {
			return httpx.JSON.InternalServerError("Unable to find attachment"), nil
		}
		if attachment == nil {
			return httpx.JSON.NotFound("Attachment not found"), nil
		}

		// Construct the object key
		key := path.Join("attachments", id)

		// WriteTo presigned URL
		presignedURL, err := s.client.PresignedGetObject(
			r.Context(),
			s.config.Bucket,
			key,
			1*time.Hour,
			nil, // No additional query parameters
		)
		if err != nil {
			return httpx.JSON.InternalServerError("Failed to generate presigned URL"), nil
		}

		// Redirect to presigned URL
		return httpx.NewRedirectResponse(http.StatusTemporaryRedirect, presignedURL.String()), nil
	})
}

// Helper function to verify if a bucket exists and is accessible
func (s *S3Storage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.config.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.config.Bucket, minio.MakeBucketOptions{
			Region: s.config.Region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}
