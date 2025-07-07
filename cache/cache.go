package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/alexisvisco/goframe/db/dbutil"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Cache persists encoded values in the database.
// Events are emitted through PostgreSQL notifications.
type Cache struct {
	db  *gorm.DB
	dsn string
}

var _ contracts.Cache = (*Cache)(nil)

// Encode encodes a value before storing it in the database. It defaults to gob.
var Encode = func(v any) ([]byte, error) {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Decode decodes a value fetched from the database. It defaults to gob.
var Decode = func(data []byte, out any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(out)
}

// SetCodec allows overriding the encode/decode functions.
func SetCodec(enc func(any) ([]byte, error), dec func([]byte, any) error) {
	if enc != nil {
		Encode = enc
	}
	if dec != nil {
		Decode = dec
	}
}

// NewCache creates a cache implementation using the given database.
func NewCache(db *gorm.DB, dsn string) *Cache {
	return &Cache{db: db, dsn: dsn}
}

func (c *Cache) Get(ctx context.Context, key string, resultPtr any) error {
	var entry coretypes.CacheEntry
	err := dbutil.DB(ctx, c.db).
		Where("expires_at IS NULL OR expires_at > now()").
		First(&entry, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("get cache entry: %w", err)
	}

	if resultPtr == nil {
		return nil
	}

	if err := Decode(entry.Value, resultPtr); err != nil {
		return fmt.Errorf("decode value: %w", err)
	}
	return nil
}

func (c *Cache) Put(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error {
	opt := &coretypes.CacheOptions{}
	for _, o := range opts {
		o(opt)
	}
	ttl := opt.TTL
	data, err := Encode(value)
	if err != nil {
		return fmt.Errorf("encode value: %w", err)
	}

	entry := coretypes.CacheEntry{
		Key:   key,
		Value: data,
	}
	if ttl > 0 {
		now := time.Now().Add(ttl)
		entry.ExpiresAt = &now
	}

	return dbutil.DB(ctx, c.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, UpdateAll: true}).
		Create(&entry).Error
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return dbutil.DB(ctx, c.db).Delete(&coretypes.CacheEntry{}, "key = ?", key).Error
}

func (c *Cache) Update(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error {
	opt := &coretypes.CacheOptions{}
	for _, o := range opts {
		o(opt)
	}
	ttl := opt.TTL

	data, err := Encode(value)
	if err != nil {
		return fmt.Errorf("encode value: %w", err)
	}

	updates := map[string]any{"value": data}
	if ttl > 0 {
		updates["expires_at"] = time.Now().Add(ttl)
	}

	result := dbutil.DB(ctx, c.db).Model(&coretypes.CacheEntry{}).
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

func (c *Cache) Watch(ctx context.Context, key string) (<-chan coretypes.CacheEvent, error) {
	ch := make(chan coretypes.CacheEvent)
	listener := pq.NewListener(c.dsn, 10*time.Second, time.Minute, nil)
	if err := listener.Listen("cache_events"); err != nil {
		return nil, err
	}

	go func() {
		defer close(ch)
		defer listener.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case n := <-listener.Notify:
				if n == nil {
					continue
				}
				var ev coretypes.CacheEvent
				if err := json.Unmarshal([]byte(n.Extra), &ev); err == nil {
					if key == "" || ev.Key == key {
						ch <- ev
					}
				}
			}
		}
	}()

	return ch, nil
}
