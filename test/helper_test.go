package integration_test

import (
	"testing"

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
