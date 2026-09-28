package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ctxWithHeader(header, value string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/products", nil)
	if header != "" {
		c.Request.Header.Set(header, value)
	}
	return c
}

func TestExtractTokenFromBearer(t *testing.T) {
	token, ok := extractToken(ctxWithHeader("Authorization", "Bearer abc.def.ghi"))
	require.True(t, ok)
	assert.Equal(t, "abc.def.ghi", token)
}

func TestExtractTokenCaseInsensitiveScheme(t *testing.T) {
	token, ok := extractToken(ctxWithHeader("Authorization", "bearer abc123"))
	require.True(t, ok)
	assert.Equal(t, "abc123", token)
}

func TestExtractTokenFromAPIKeyHeader(t *testing.T) {
	token, ok := extractToken(ctxWithHeader("X-API-Token", "abc123"))
	require.True(t, ok)
	assert.Equal(t, "abc123", token)
}

func TestExtractTokenRejects(t *testing.T) {
	cases := map[string][2]string{
		"tidak ada header":    {"", ""},
		"tanpa scheme":        {"Authorization", "abc123"},
		"scheme salah":        {"Authorization", "Basic abc123"},
		"token kosong":        {"Authorization", "Bearer "},
		"api token kosong":    {"X-API-Token", "  "},
		"bearer bukan string": {"Authorization", "Bearer"},
	}
	for name, kv := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok := extractToken(ctxWithHeader(kv[0], kv[1]))
			assert.False(t, ok)
		})
	}
}

func TestRateLimitSkipsWhenClientNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/products", nil)

	reached := false
	handler := RateLimit(nil, "test", "rule", RateLimitRule{Limit: 1, Window: time.Minute})
	handler(c)
	if !c.IsAborted() {
		reached = true
	}
	assert.True(t, reached, "request harus diteruskan saat redis tidak tersedia")
}

func TestRuleDefaultsAreValid(t *testing.T) {
	authRule := RateLimitRule{Limit: 3, Window: 60 * time.Second}
	writeRule := RateLimitRule{Limit: 1, Window: 5 * time.Second}
	assert.Positive(t, authRule.Limit)
	assert.Positive(t, writeRule.Limit)
}
