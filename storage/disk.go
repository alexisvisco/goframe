package storage

import (
	"context"
	"net/http"

	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
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
	//TODO implement me
	panic("implement me")
}

func (d DiskStorage) DeleteAttachment(ctx context.Context, id string) error {
	//TODO implement me
	panic("implement me")
}

func (d DiskStorage) AttachmentHandler(pathValueField string) http.HandlerFunc {
	//TODO implement me
	panic("implement me")
}
