package service

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restapi-test/internal/config"
	"restapi-test/internal/model"
	"restapi-test/internal/repository"
)

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:      "test-secret-minimal-32-karakter-panjang",
		AccessTokenTTL: 15 * time.Minute,
		RefreshTTL:     30 * 24 * time.Hour,
		BcryptCost:     4,
	}
}

func TestParseAccessTokenValid(t *testing.T) {
	svc := &AuthService{cfg: testConfig()}
	claims := Claims{
		UserID:   7,
		Username: "jhon_doe",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "7",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(svc.cfg.JWTSecret))
	require.NoError(t, err)

	parsed, err := svc.ParseAccessToken(raw)
	require.NoError(t, err)
	assert.Equal(t, uint(7), parsed.UserID)
	assert.Equal(t, "jhon_doe", parsed.Username)
}

func TestParseAccessTokenRejected(t *testing.T) {
	svc := &AuthService{cfg: testConfig()}
	cfg := svc.cfg

	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	}).SignedString([]byte(cfg.JWTSecret))
	require.NoError(t, err)

	noExp, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{}).SignedString([]byte(cfg.JWTSecret))
	require.NoError(t, err)

	wrongSecret, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}).SignedString([]byte("secret-lain-yang-panjang-sekali-12345"))
	require.NoError(t, err)

	noneAlg, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	for name, raw := range map[string]string{
		"kedaluwarsa":    expired,
		"tanpa exp":      noExp,
		"secret salah":   wrongSecret,
		"alg none":       noneAlg,
		"string acak":    "bukan.token.jwt",
		"string kosong":  "",
		"tabrakan freak": "a.b",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.ParseAccessToken(raw)
			assert.Error(t, err)
		})
	}
}

func TestParseAccessTokenExpiredError(t *testing.T) {
	svc := &AuthService{cfg: testConfig()}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Second))},
	}).SignedString([]byte(svc.cfg.JWTSecret))
	require.NoError(t, err)

	_, err = svc.ParseAccessToken(raw)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTokenExpired), "harus melaporkan token kedaluwarsa")
}

func TestHashTokenIsStableAndHidden(t *testing.T) {
	raw := "rahasia-token"
	first := hashToken(raw)
	assert.Equal(t, first, hashToken(raw))
	assert.Len(t, first, 64)
	assert.NotEqual(t, raw, first)
}

func TestRandomTokenUnique(t *testing.T) {
	a, err := randomToken(32)
	require.NoError(t, err)
	b, err := randomToken(32)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.Greater(t, len(a), 32)
}

func TestRegisterPasswordMismatch(t *testing.T) {
	svc := NewAuthService(repository.NewUserRepository(nil), repository.NewTokenRepository(nil), testConfig())
	_, err := svc.Register(t.Context(), "jhon_doe", "supersecret", "beda")
	assert.ErrorIs(t, err, ErrPasswordMismatch)
}

func TestRegisterUsernamePattern(t *testing.T) {
	valid := []string{"jhon_doe", "jhon.doe", "jhon-doe", "user123", "ab1"}
	for _, name := range valid {
		assert.True(t, usernamePattern.MatchString(name), name)
	}

	invalid := []string{"ab", "jhon doe", "jhon@doe", "jhon/doe", "user+1", string(make([]byte, 31))}
	for _, name := range invalid {
		assert.False(t, usernamePattern.MatchString(name), name)
	}
}

func TestRegisterRejectsBadUsername(t *testing.T) {
	svc := NewAuthService(repository.NewUserRepository(nil), repository.NewTokenRepository(nil), testConfig())
	_, err := svc.Register(t.Context(), "ab", "supersecret", "supersecret")
	assert.ErrorIs(t, err, ErrUsernameInvalid)
}

func TestBuildMeta(t *testing.T) {
	meta := buildMeta(42, 2, 10)
	assert.Equal(t, int64(42), meta.Total)
	assert.Equal(t, 5, meta.TotalPages)
	assert.True(t, meta.HasNext)
	assert.True(t, meta.HasPrevious)

	first := buildMeta(42, 1, 10)
	assert.False(t, first.HasPrevious)
	assert.True(t, first.HasNext)

	last := buildMeta(42, 5, 10)
	assert.False(t, last.HasNext)

	empty := buildMeta(0, 1, 10)
	assert.Equal(t, 0, empty.TotalPages)
	assert.False(t, empty.HasNext)
}

func TestToProductResponseNeverReturnsNilImages(t *testing.T) {
	out := toProductResponse(model.Product{ID: 1, Title: "x"})
	assert.NotNil(t, out.Images)
	assert.Empty(t, out.Images)
}
