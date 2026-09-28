package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restapi-test/internal/config"
	"restapi-test/internal/response"
	"restapi-test/internal/service"
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

func TestExtractTokenRejects(t *testing.T) {
	cases := map[string][2]string{
		"tidak ada header":    {"", ""},
		"tanpa scheme":        {"Authorization", "abc123"},
		"scheme salah":        {"Authorization", "Basic abc123"},
		"token kosong":        {"Authorization", "Bearer "},
		"api token ditolak":   {"X-API-Token", "abc123"},
		"bearer bukan string": {"Authorization", "Bearer"},
		"prefix bearer":       {"Authorization", "BearerXyz"},
	}
	for name, kv := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok := extractToken(ctxWithHeader(kv[0], kv[1]))
			assert.False(t, ok)
		})
	}
}

func TestAuthRejectsRequestWithoutToken(t *testing.T) {
	engine := authEngine()

	for _, path := range []string{"/api/products", "/api/products/1"} {
		rec := doGet(engine, path, "")
		assert.Equal(t, 401, rec.Code, path)
		assert.Contains(t, rec.Body.String(), "UNAUTHORIZED", path)
	}
}

func TestAuthRejectsGarbageToken(t *testing.T) {
	rec := doGet(authEngine(), "/api/products", "token.palsu")

	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), "TOKEN_INVALID")
}

func TestAuthRejectsLegacyXAPITokenHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	req.Header.Set("X-API-Token", "token.lama")

	rec := httptest.NewRecorder()
	authEngine().ServeHTTP(rec, req)

	assert.Equal(t, 401, rec.Code, "X-API-Token tidak lagi menjadi alternatif yang sah")
	assert.Contains(t, rec.Body.String(), "UNAUTHORIZED")
}

func TestAuthPassesValidToken(t *testing.T) {
	svc := svcForTest()
	rec := doGet(authEngine(), "/api/products", validToken(t, svc))

	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "jhon_doe", rec.Body.String())
}

func TestAuthRejectsExpiredToken(t *testing.T) {
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, service.Claims{
		UserID:   1,
		Username: "jhon_doe",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}).SignedString([]byte(testSecret))
	require.NoError(t, err)

	rec := doGet(authEngine(), "/api/products", raw)

	assert.Equal(t, 401, rec.Code)
	assert.Contains(t, rec.Body.String(), "TOKEN_EXPIRED")
}

func TestTokenErrorCodeMapping(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"expired": {service.ErrTokenExpired, response.CodeTokenExpired},
		"revoked": {service.ErrTokenRevoked, response.CodeTokenRevoked},
		"invalid": {service.ErrTokenInvalid, response.CodeTokenInvalid},
		"lainnya": {errors.New("koneksi gagal"), response.CodeTokenInvalid},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tokenErrorCode(tc.err))
		})
	}
}

const testSecret = "test-secret-minimal-32-karakter-panjang"

func svcForTest() *service.AuthService {
	return service.NewAuthService(nil, nil, &config.Config{JWTSecret: testSecret})
}

func validToken(t *testing.T, svc *service.AuthService) string {
	t.Helper()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, service.Claims{
		UserID:   1,
		Username: "jhon_doe",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString([]byte(testSecret))
	require.NoError(t, err)
	return raw
}

func authEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Auth(svcForTest()))
	engine.GET("/api/products", func(c *gin.Context) {
		actor, _ := CurrentUser(c)
		c.String(200, actor.Username)
	})
	engine.GET("/api/products/:id", func(c *gin.Context) {
		actor, _ := CurrentUser(c)
		c.String(200, actor.Username)
	})
	return engine
}

func doGet(engine *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}
