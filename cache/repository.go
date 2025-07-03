package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/alexisvisco/goframe/db/dbutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

var _ contracts.CacheRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, key string, resultPtr any) error {
	var entry coretypes.CacheEntry
	err := dbutil.DB(ctx, r.db).
		Where("expires_at IS NULL OR expires_at > now()").
		First(&entry, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Key not found
		}
		return fmt.Errorf("get cache entry: %w", err)
	}

	if err := json.Unmarshal(entry.Value, &resultPtr); err != nil {
		return fmt.Errorf("unmarshal value: %w", err)
	}

	return nil
}

func (r *Repository) Put(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error {
	opt := &coretypes.CacheOptions{}
	for _, o := range opts {
		o(opt)
	}
	ttl := opt.TTL
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	entry := coretypes.CacheEntry{
		Key:   key,
		Value: data,
	}
	if ttl > 0 {
		now := time.Now().Add(ttl)
		entry.ExpiresAt = &now
	}

	return dbutil.DB(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, UpdateAll: true}).
		Create(&entry).Error
}

func (r *Repository) Delete(ctx context.Context, key string) error {
	return dbutil.DB(ctx, r.db).Delete(&coretypes.CacheEntry{}, "key = ?", key).Error
}

func (r *Repository) Update(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error {
	opt := &coretypes.CacheOptions{}
	for _, o := range opts {
		o(opt)
	}
	ttl := opt.TTL

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	updates := map[string]any{"value": data}
	if ttl > 0 {
		updates["expires_at"] = time.Now().Add(ttl)
	}

	result := dbutil.DB(ctx, r.db).Model(&coretypes.CacheEntry{}).
		Where("key = ?", key).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("cache key not found")
	}
	return nil
}
