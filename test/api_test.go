package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restapi-test/internal/cache"
	"restapi-test/internal/config"
	"restapi-test/internal/database"
	"restapi-test/internal/handler"
	"restapi-test/internal/middleware"
	"restapi-test/internal/repository"
	"restapi-test/internal/router"
	"restapi-test/internal/service"
)

type testEnv struct {
	router http.Handler
	db     *gorm.DB
	redis  *redis.Client
	cfg    *config.Config
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL belum diisi, test integrasi dilewati")
	}
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("TEST_REDIS_ADDR belum diisi, test integrasi dilewati")
	}

	cfg := &config.Config{
		GinMode:         gin.TestMode,
		DatabaseURL:     dsn,
		RedisAddr:       redisAddr,
		RedisPassword:   os.Getenv("TEST_REDIS_PASSWORD"),
		JWTSecret:       "test-secret-minimal-32-karakter-panjang",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTTL:      24 * time.Hour,
		BcryptCost:      4,
		CacheTTLList:    time.Minute,
		CacheTTLItem:    5 * time.Minute,
		ConnectAttempts: 1,
		ConnectBackoff:  10 * time.Millisecond,
		RateLimitPrefix: "rl:run:" + fmt.Sprint(time.Now().UnixNano()),
	}
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(cfg)
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())
	truncateAll(t, db)

	productCache := cache.New(cfg)
	require.True(t, productCache.Healthy(), "redis harus terjangkau untuk test integrasi")

	productRepo := repository.NewProductRepository(db)
	authService := service.NewAuthService(repository.NewUserRepository(db), repository.NewTokenRepository(db), cfg)
	productService := service.NewProductService(productRepo, productCache, cfg.CacheTTLList, cfg.CacheTTLItem)

	engine := router.New(router.Deps{
		Redis:           productCache.Client(),
		RateLimitPrefix: cfg.RateLimitPrefix,
		Products:        handler.NewProductHandler(productService),
		Auth:            handler.NewAuthHandler(authService),
		AuthMiddleware:  middleware.Auth(authService),
	})

	t.Cleanup(func() {
		truncateAll(t, db)
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer flushCancel()
		if client := productCache.Client(); client != nil {
			_ = client.FlushDB(flushCtx).Err()
		}
		_ = productCache.Close()
		_ = sqlDB.Close()
	})

	return &testEnv{router: engine, db: db, redis: productCache.Client(), cfg: cfg}
}

type authResult struct {
	Access  string
	Refresh string
}

func (e *testEnv) request(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewBuffer(raw)
	} else {
		reader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) registerAndLogin(t *testing.T, username string) authResult {
	t.Helper()

	reg := e.request(t, http.MethodPost, "/api/auth/register", map[string]any{
		"username":              username,
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}, "")
	require.Equal(t, http.StatusCreated, reg.Code, reg.Body.String())

	login := e.request(t, http.MethodPost, "/api/auth/login", map[string]any{
		"username": username,
		"password": "supersecret",
	}, "")
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())

	var payload struct {
		AuthenticationToken string `json:"authentication_token"`
		RefreshToken        string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &payload))
	require.NotEmpty(t, payload.AuthenticationToken)
	require.NotEmpty(t, payload.RefreshToken)

	return authResult{Access: payload.AuthenticationToken, Refresh: payload.RefreshToken}
}

func (e *testEnv) createProduct(t *testing.T, token string, payload map[string]any) map[string]any {
	t.Helper()
	rec := e.request(t, http.MethodPost, "/api/products", payload, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var out struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	e.clearRateLimit(t)
	return out.Data
}

func defaultProduct() map[string]any {
	return map[string]any{
		"title":       "Awesome T-Shirt",
		"price":       99.99,
		"description": "High-quality cotton t-shirt",
		"category":    "Clothes",
		"images":      []string{"https://placeimg.com/640/480/any"},
	}
}

func TestHealth(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodGet, "/health", nil, "")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterAndLoginFlow(t *testing.T) {
	env := newTestEnv(t)
	username := "user_" + fmt.Sprint(time.Now().UnixNano())
	auth := env.registerAndLogin(t, username)
	check(t, auth.Access != "", "access token harus ada")
}

func TestRegisterStoresHashedPassword(t *testing.T) {
	env := newTestEnv(t)
	username := "hash_" + fmt.Sprint(time.Now().UnixNano())

	rec := env.request(t, http.MethodPost, "/api/auth/register", map[string]any{
		"username":              username,
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}, "")
	require.Equal(t, http.StatusCreated, rec.Code)
	check(t, !bytes.Contains(rec.Body.Bytes(), []byte("supersecret")), "password tidak boleh muncul di respons")

	var stored string
	row := env.db.Raw("SELECT password_hash FROM users WHERE username = ?", username).Row()
	require.NoError(t, row.Scan(&stored))
	check(t, stored != "", "password_hash harus tersimpan")
	check(t, stored != "supersecret", "password harus di-hash")
	check(t, len(stored) > 20, "hash bcrypt harus panjang")
}

func TestRegisterDuplicateUsername(t *testing.T) {
	env := newTestEnv(t)
	username := "dup_" + fmt.Sprint(time.Now().UnixNano())
	payload := map[string]any{
		"username":              username,
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}

	first := env.request(t, http.MethodPost, "/api/auth/register", payload, "")
	require.Equal(t, http.StatusCreated, first.Code)

	second := env.request(t, http.MethodPost, "/api/auth/register", payload, "")
	require.Equal(t, http.StatusConflict, second.Code)
	check(t, bytes.Contains(second.Body.Bytes(), []byte("CONFLICT")), "respons duplikasi username harus kode CONFLICT")
}

func TestRegisterPasswordMismatch(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodPost, "/api/auth/register", map[string]any{
		"username":              "mismatch_" + fmt.Sprint(time.Now().UnixNano()),
		"password":              "supersecret",
		"password_confirmation": "lain12345",
	}, "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginWrongPassword(t *testing.T) {
	env := newTestEnv(t)
	username := "wrong_" + fmt.Sprint(time.Now().UnixNano())
	env.request(t, http.MethodPost, "/api/auth/register", map[string]any{
		"username":              username,
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}, "")

	rec := env.request(t, http.MethodPost, "/api/auth/login", map[string]any{
		"username": username,
		"password": "salahbanget",
	}, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginUnknownUser(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "tidak_ada_user_ini",
		"password": "supersecret",
	}, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefreshRotatesToken(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "refresh_"+fmt.Sprint(time.Now().UnixNano()))

	rec := env.request(t, http.MethodPost, "/api/auth/refresh", map[string]any{
		"refresh_token": auth.Refresh,
	}, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var payload struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	check(t, payload.Data.RefreshToken != "" && payload.Data.RefreshToken != auth.Refresh, "refresh token harus berganti")

	env.clearRateLimit(t)
	reuse := env.request(t, http.MethodPost, "/api/auth/refresh", map[string]any{
		"refresh_token": auth.Refresh,
	}, "")
	require.Equal(t, http.StatusUnauthorized, reuse.Code, "token lama harus dicabut")
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "logout_"+fmt.Sprint(time.Now().UnixNano()))

	rec := env.request(t, http.MethodPost, "/api/auth/logout", map[string]any{
		"refresh_token": auth.Refresh,
	}, "")
	require.Equal(t, http.StatusOK, rec.Code)

	env.clearRateLimit(t)
	after := env.request(t, http.MethodPost, "/api/auth/refresh", map[string]any{
		"refresh_token": auth.Refresh,
	}, "")
	require.Equal(t, http.StatusUnauthorized, after.Code)
}

func TestCreateProductRequiresAuth(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodPost, "/api/products", defaultProduct(), "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateProductRejectsBadToken(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodPost, "/api/products", defaultProduct(), "token.palsu")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateProductValidation(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "valid_"+fmt.Sprint(time.Now().UnixNano()))

	cases := map[string]map[string]any{
		"tanpa title":    {"price": 10, "category": "Clothes", "images": []string{"https://a.test/1.png"}},
		"tanpa price":    {"title": "x", "category": "Clothes", "images": []string{"https://a.test/1.png"}},
		"tanpa category": {"title": "x", "price": 10, "images": []string{"https://a.test/1.png"}},
		"tanpa images":   {"title": "x", "price": 10, "category": "Clothes"},
		"images kosong":  {"title": "x", "price": 10, "category": "Clothes", "images": []string{}},
		"price negatif":  {"title": "x", "price": -1, "category": "Clothes", "images": []string{"https://a.test/1.png"}},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			env.clearRateLimit(t)
			rec := env.request(t, http.MethodPost, "/api/products", payload, auth.Access)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			check(t, bytes.Contains(rec.Body.Bytes(), []byte("VALIDATION_ERROR")), "validasi gagal harus kode VALIDATION_ERROR")
		})
	}
}

func TestProductCRUDLifecycle(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "crud_"+fmt.Sprint(time.Now().UnixNano()))

	created := env.createProduct(t, auth.Access, defaultProduct())
	id, _ := created["id"].(float64)
	check(t, id > 0, "produk harus punya id")
	check(t, created["created_by"] != nil, "created_by harus terisi")
	check(t, created["created_by_id"] != nil, "created_by_id harus terisi")
	check(t, created["updated_at"] != nil, "updated_at harus terisi")

	got := env.request(t, http.MethodGet, fmt.Sprintf("/api/products/%d", int64(id)), nil, "")
	require.Equal(t, http.StatusOK, got.Code)

	update := env.request(t, http.MethodPut, fmt.Sprintf("/api/products/%d", int64(id)), map[string]any{
		"title": "Awesome T-Shirt v2",
		"price": 149.99,
	}, auth.Access)
	require.Equal(t, http.StatusOK, update.Code, update.Body.String())
	check(t, bytes.Contains(update.Body.Bytes(), []byte("Awesome T-Shirt v2")), "hasil update harus memuat judul baru")
	check(t, bytes.Contains(update.Body.Bytes(), []byte("Clothes")), "field yang tidak dikirim harus dipertahankan")

	env.clearRateLimit(t)
	del := env.request(t, http.MethodDelete, fmt.Sprintf("/api/products/%d", int64(id)), nil, auth.Access)
	require.Equal(t, http.StatusOK, del.Code, del.Body.String())

	gone := env.request(t, http.MethodGet, fmt.Sprintf("/api/products/%d", int64(id)), nil, "")
	require.Equal(t, http.StatusNotFound, gone.Code)
	check(t, bytes.Contains(gone.Body.Bytes(), []byte("NOT_FOUND")), "produk terhapus harus 404 dengan kode NOT_FOUND")
}

func TestGetProductNotFound(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodGet, "/api/products/99999999", nil, "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	check(t, bytes.Contains(rec.Body.Bytes(), []byte("produk tidak ditemukan")), "pesan 404 produk harus sesuai")
}

func TestGetProductInvalidID(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodGet, "/api/products/abc", nil, "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWriteEndpointsRequireAuth(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "guard_"+fmt.Sprint(time.Now().UnixNano()))
	created := env.createProduct(t, auth.Access, defaultProduct())
	id := int64(created["id"].(float64))

	require.Equal(t, http.StatusUnauthorized, env.request(t, http.MethodPut, fmt.Sprintf("/api/products/%d", id), map[string]any{"price": 10}, "").Code)
	require.Equal(t, http.StatusUnauthorized, env.request(t, http.MethodDelete, fmt.Sprintf("/api/products/%d", id), nil, "").Code)
}

func TestListFilterSearchCategoryPagination(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "list_"+fmt.Sprint(time.Now().UnixNano()))

	env.createProduct(t, auth.Access, map[string]any{"title": "Kaos Katun", "price": 50, "category": "Clothes", "images": []string{"https://a.test/1.png"}})
	env.createProduct(t, auth.Access, map[string]any{"title": "Celana Jeans", "price": 150, "category": "Clothes", "images": []string{"https://a.test/2.png"}})
	env.createProduct(t, auth.Access, map[string]any{"title": "Mouse Nirkabel", "price": 45, "category": "Electronics", "images": []string{"https://a.test/3.png"}})

	listAll := env.request(t, http.MethodGet, "/api/products?limit=100", nil, "")
	require.Equal(t, http.StatusOK, listAll.Code, listAll.Body.String())

	var all struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(listAll.Body.Bytes(), &all))
	require.GreaterOrEqual(t, len(all.Data), 3)
	require.GreaterOrEqual(t, all.Meta.Total, 3)

	search := env.request(t, http.MethodGet, "/api/products?search=Kaos&limit=100", nil, "")
	var searched struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(search.Body.Bytes(), &searched))
	require.Len(t, searched.Data, 1)
	check(t, searched.Data[0]["title"] == "Kaos Katun", "judul hasil search harus Kaos Katun")

	byCategory := env.request(t, http.MethodGet, "/api/products?category=Electronics&limit=100", nil, "")
	var filtered struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(byCategory.Body.Bytes(), &filtered))
	require.Len(t, filtered.Data, 1)
	check(t, filtered.Data[0]["category"] == "Electronics", "kategori hasil filter harus Electronics")

	page1 := env.request(t, http.MethodGet, "/api/products?limit=2&page=1", nil, "")
	var first struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Page       int  `json:"page"`
			Limit      int  `json:"limit"`
			TotalPages int  `json:"total_pages"`
			HasNext    bool `json:"has_next"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(page1.Body.Bytes(), &first))
	require.Len(t, first.Data, 2)
	check(t, first.Meta.Page == 1 && first.Meta.Limit == 2, "meta page dan limit harus sesuai")
	check(t, first.Meta.HasNext, "halaman pertama harus punya halaman berikutnya")

	page2 := env.request(t, http.MethodGet, "/api/products?limit=2&page=2", nil, "")
	var second struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(page2.Body.Bytes(), &second))
	require.NotEmpty(t, second.Data)
	check(t, second.Data[0]["id"] != first.Data[0]["id"], "halaman berbeda harus berisi data berbeda")
}

func TestListRejectsBadQuery(t *testing.T) {
	env := newTestEnv(t)
	for _, path := range []string{"/api/products?page=0", "/api/products?page=abc", "/api/products?limit=0", "/api/products?limit=999"} {
		rec := env.request(t, http.MethodGet, path, nil, "")
		require.Equal(t, http.StatusBadRequest, rec.Code, path)
	}
}

func TestCORSHeaders(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodGet, "/api/products", nil, "")
	check(t, rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Vary") == "Origin", "header CORS harus ada")
}

func TestCORSPreflight(t *testing.T) {
	env := newTestEnv(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
	req.Header.Set("Origin", "https://frontend.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	check(t, rec.Code == http.StatusNoContent, "preflight harus 204")
	check(t, rec.Header().Get("Access-Control-Allow-Origin") == "https://frontend.example", "preflight harus mengembalikan origin pemanggil")
	check(t, bytes.Contains([]byte(rec.Header().Get("Access-Control-Allow-Methods")), []byte("DELETE")), "preflight harus mengizinkan DELETE")
}

func TestProductWriteRateLimit(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "rl_"+fmt.Sprint(time.Now().UnixNano()))

	first := env.request(t, http.MethodPost, "/api/products", defaultProduct(), auth.Access)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	second := env.request(t, http.MethodPost, "/api/products", defaultProduct(), auth.Access)
	require.Equal(t, http.StatusTooManyRequests, second.Code, "permintaan kedua dalam 5 detik harus diblokir")
	check(t, bytes.Contains(second.Body.Bytes(), []byte("RATE_LIMITED")), "rate limit harus kode RATE_LIMITED")
	check(t, second.Header().Get("Retry-After") != "", "header Retry-After wajib ada")
}

func TestAuthRateLimit(t *testing.T) {
	env := newTestEnv(t)
	username := "authrl_" + fmt.Sprint(time.Now().UnixNano())
	payload := map[string]any{
		"username":              username,
		"password":              "supersecret",
		"password_confirmation": "supersecret",
	}

	created := env.request(t, http.MethodPost, "/api/auth/register", payload, "")
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())

	for i := 0; i < 2; i++ {
		rec := env.request(t, http.MethodPost, "/api/auth/login", map[string]any{
			"username": username,
			"password": "supersecret",
		}, "")
		require.Equal(t, http.StatusOK, rec.Code, "percobaan %d", i+2)
	}

	blocked := env.request(t, http.MethodPost, "/api/auth/login", map[string]any{
		"username": username,
		"password": "supersecret",
	}, "")
	require.Equal(t, http.StatusTooManyRequests, blocked.Code, "percobaan ke-4 harus diblokir")
}

func TestCacheIsUsedAndInvalidated(t *testing.T) {
	env := newTestEnv(t)
	auth := env.registerAndLogin(t, "cache_"+fmt.Sprint(time.Now().UnixNano()))

	env.request(t, http.MethodGet, "/api/products?category=CacheTest&limit=100", nil, "")
	keys := env.redisKeys(t, "products:v*")
	check(t, len(keys) > 0, "cache list harus tersimpan di redis")

	env.createProduct(t, auth.Access, map[string]any{
		"title": "Cache Buster", "price": 10, "category": "CacheTest",
		"images": []string{"https://a.test/c.png"},
	})

	rec := env.request(t, http.MethodGet, "/api/products?category=CacheTest&limit=100", nil, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	check(t, bytes.Contains(rec.Body.Bytes(), []byte("Cache Buster")), "data baru harus terlihat setelah invalidasi cache")
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	env := newTestEnv(t)
	rec := env.request(t, http.MethodGet, "/api/tidak-ada", nil, "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	check(t, bytes.Contains(rec.Body.Bytes(), []byte("NOT_FOUND")), "route tak dikenal harus kode NOT_FOUND")
}
