package coretypes

import (
	"io"
	"time"
)

type (
	Attachment struct {
		ID          string     `json:"id"`
		Filename    string     `json:"filename"`
		ContentType string     `json:"content_type"`
		ByteSize    int64      `json:"byte_size"`
		Key         string     `json:"key"`
		Checksum    string     `json:"checksum"`
		CreatedAt   time.Time  `json:"created_at"`
		DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	}

	UploadAttachmentOptions struct {
		Filename            string
		Content             io.Reader
		CurrentAttachmentID *string // Optional, for replacing existing attachment
	}
)
