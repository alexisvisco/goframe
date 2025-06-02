package contracts

import (
	"context"
	"net/http"

	"github.com/alexisvisco/goframe/core/coretypes"
)

type (
	StorageRepository interface {
		SaveAttachment(ctx context.Context, attachment *coretypes.Attachment) error
		DeleteAttachment(ctx context.Context, id string, cleanup func() error) error
		GetAttachment(ctx context.Context, id string) (*coretypes.Attachment, error)
	}

	Storage interface {
		UploadAttachment(ctx context.Context, opts coretypes.UploadAttachmentOptions) (*coretypes.Attachment, error)
		DeleteAttachment(ctx context.Context, id string) error
		AttachmentHandler(pathValueField string) http.HandlerFunc
	}
)
