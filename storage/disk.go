package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/http/httpx"
	"github.com/gabriel-vasile/mimetype"
	"github.com/nrednav/cuid2"
)

type DiskStorage struct {
	repository contracts.StorageRepository
	cfg        configuration.Storage
}

var _ contracts.Storage = (*DiskStorage)(nil)

func NewDiskStorage(cfg configuration.Storage, repository contracts.StorageRepository) *DiskStorage {
	return &DiskStorage{
		repository: repository,
		cfg:        cfg,
	}
}

func (d DiskStorage) UploadAttachment(ctx context.Context, opts coretypes.UploadAttachmentOptions) (*coretypes.Attachment, error) {
	// generate ID
	id := cuid2.Generate()
	if opts.CurrentAttachmentID != nil {
		id = *opts.CurrentAttachmentID
	}

	// ensure base directory exists
	baseDir := filepath.Join(d.cfg.Directory, "attachments")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmpFile, hash), opts.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	buffer := make([]byte, 3072)
	_, err = tmpFile.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to read temp file: %w", err)
	}
	contentType := mimetype.Detect(buffer)

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	key := filepath.Join("attachments", fmt.Sprintf("%s-%s", id, str.Slugify(opts.Filename)))
	destPath := filepath.Join(d.cfg.Directory, key)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	if _, err := io.Copy(destFile, tmpFile); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	attachment := &coretypes.Attachment{
		ID:          id,
		Filename:    opts.Filename,
		ContentType: contentType.String(),
		ByteSize:    size,
		Key:         key,
		Checksum:    hex.EncodeToString(hash.Sum(nil)),
		CreatedAt:   time.Now(),
	}

	if err := d.repository.SaveAttachment(ctx, attachment); err != nil {
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	return attachment, nil
}

func (d DiskStorage) DeleteAttachment(ctx context.Context, id string) error {
	attachment, err := d.repository.GetAttachment(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch attachment: %w", err)
	}

	if attachment == nil {
		return nil
	}

	filePath := filepath.Join(d.cfg.Directory, attachment.Key)

	err = d.repository.DeleteAttachment(ctx, id, func() error {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

func (d DiskStorage) AttachmentHandler(pathValueField string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue(pathValueField)
		if id == "" {
			_ = httpx.JSON.BadRequest("Attachment ID is required").WriteTo(w, r)
			return
		}

		attachment, err := d.repository.GetAttachment(r.Context(), id)
		if err != nil {
			_ = httpx.JSON.InternalServerError("Unable to find attachment").WriteTo(w, r)
			return
		}
		if attachment == nil {
			_ = httpx.JSON.NotFound("Attachment not found").WriteTo(w, r)
			return
		}

		filePath := filepath.Join(d.cfg.Directory, attachment.Key)
		f, err := os.Open(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				_ = httpx.JSON.NotFound("Attachment not found").WriteTo(w, r)
				return
			}
			_ = httpx.JSON.InternalServerError("Failed to open attachment").WriteTo(w, r)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", attachment.ContentType)
		http.ServeContent(w, r, attachment.Filename, attachment.CreatedAt, f)
	}
}
