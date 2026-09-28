package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func check(t *testing.T, condition bool, msg string, args ...any) {
	t.Helper()
	if !condition {
		if len(args) > 0 {
			t.Errorf("gagal: "+msg, args...)
			return
		}
		t.Errorf("gagal: %s", msg)
	}
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec(`TRUNCATE TABLE products, refresh_tokens, users RESTART IDENTITY CASCADE`).Error
	require.NoError(t, err)
}

func (e *testEnv) redisKeys(t *testing.T, pattern string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var found []string
	var cursor uint64
	for {
		keys, next, err := e.redis.Scan(ctx, cursor, pattern, 100).Result()
		require.NoError(t, err)
		found = append(found, keys...)
		if next == 0 {
			break
		}
		cursor = next
	}
	return found
}

func (e *testEnv) clearRateLimit(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys := e.redisKeys(t, e.cfg.RateLimitPrefix+":*")
	if len(keys) == 0 {
		return
	}
	require.NoError(t, e.redis.Del(ctx, keys...).Err())
}
