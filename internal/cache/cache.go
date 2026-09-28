package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"restapi-test/internal/config"
)

const versionKey = "products:ver"

type Cache struct {
	client  *redis.Client
	ttlList time.Duration
	ttlItem time.Duration
	healthy atomic.Bool
}

func New(cfg *config.Config) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	c := &Cache{client: client, ttlList: cfg.CacheTTLList, ttlItem: cfg.CacheTTLItem}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("cache: redis tidak terjangkau, cache dinonaktifkan: %v", err)
		c.healthy.Store(false)
		return c
	}
	c.healthy.Store(true)
	log.Printf("cache: redis terhubung di %s", cfg.RedisAddr)
	return c
}

func (c *Cache) Healthy() bool {
	return c != nil && c.healthy.Load() && c.client != nil
}

func (c *Cache) Client() *redis.Client {
	if !c.Healthy() {
		return nil
	}
	return c.client
}

func (c *Cache) Ping(ctx context.Context) error {
	if !c.Healthy() {
		return nil
	}
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Cache) version(ctx context.Context) uint64 {
	if !c.Healthy() {
		return 0
	}
	v, err := c.client.Get(ctx, versionKey).Uint64()
	if err != nil {
		return 0
	}
	return v
}

func (c *Cache) GetJSON(ctx context.Context, key string, dest any) bool {
	if !c.Healthy() {
		return false
	}
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false
	}
	return true
}

func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if !c.Healthy() {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	if err := c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		log.Printf("cache: gagal set %s: %v", key, err)
	}
}

func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if !c.Healthy() || len(keys) == 0 {
		return
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		log.Printf("cache: gagal hapus %v: %v", keys, err)
	}
}

func (c *Cache) BumpVersion(ctx context.Context) {
	if !c.Healthy() {
		return
	}
	if err := c.client.Incr(ctx, versionKey).Err(); err != nil {
		log.Printf("cache: gagal naikkan versi: %v", err)
	}
}

func (c *Cache) ListKey(ctx context.Context, parts ...string) string {
	return c.key(ctx, "list", parts...)
}

func (c *Cache) ItemKey(ctx context.Context, id uint64) string {
	return c.key(ctx, "item", strconv.FormatUint(id, 10))
}

func (c *Cache) key(ctx context.Context, kind string, parts ...string) string {
	raw := kind
	for _, p := range parts {
		raw += "|" + p
	}
	sum := sha256.Sum256([]byte(raw))
	return "products:v" + strconv.FormatUint(c.version(ctx), 10) + ":" + kind + ":" + hex.EncodeToString(sum[:4])
}
