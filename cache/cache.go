package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Cache struct {
	repo contracts.CacheRepository
	db   *gorm.DB
	dsn  string
}

var _ contracts.Cache = (*Cache)(nil)

func NewCache(repo contracts.CacheRepository, db *gorm.DB, dsn string) *Cache {
	return &Cache{repo: repo, db: db, dsn: dsn}
}

func (c *Cache) Get(ctx context.Context, key string, resultPtr any) error {
	err := c.repo.Get(ctx, key)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) Put(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error {
	if err := c.repo.Put(ctx, key, value, opts...); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.repo.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Update(ctx context.Context, key string, value any, opts ...coretypes.CacheOption) error {
	if err := c.repo.Update(ctx, key, value, opts...); err != nil {
		return err
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
