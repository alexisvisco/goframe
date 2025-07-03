package contracts

import (
	"context"

	"github.com/alexisvisco/goframe/core/coretypes"
)

type (
	CacheRepository interface {
		Get(ctx context.Context, key string, resultPtr any) error
		Put(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error
		Delete(ctx context.Context, key string) error
		Update(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error
	}

	Cache interface {
		CacheRepository
		Watch(ctx context.Context, key string) (<-chan coretypes.CacheEvent, error)
	}
)
