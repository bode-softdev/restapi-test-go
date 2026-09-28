# TODO List — REST API Product Management

Status: `[x]` selesai · `[ ]` belum

## 0. Tooling

- [x] `git init`
- [x] `go mod init restapi-test`
- [x] Go 1.27.0, Docker 29.7.2, Compose v5.5.1, git 2.55 terverifikasi
- [x] Dependency: gin, gorm, driver/postgres, datatypes, go-redis, golang-jwt, bcrypt, godotenv, swaggo, testify

## 1. Struktur dan lapisan

- [x] `cmd/api/main.go` wiring + graceful shutdown
- [x] `internal/config` — env loader, validasi, fail-fast
- [x] `internal/model` — User, Product, RefreshToken
- [x] `internal/database` — connect dengan retry, AutoMigrate, seed idempotent
- [x] `internal/repository` — ProductRepository, UserRepository, TokenRepository
- [x] `internal/service` — ProductService, AuthService
- [x] `internal/cache` — Redis + invalidasi berbasis versi
- [x] `internal/middleware` — auth, rate limit, CORS, logger
- [x] `internal/handler` — ProductHandler, AuthHandler
- [x] `internal/response` — envelope JSON + helper error
- [x] `internal/router` — registrasi route

## 2. Model data

- [x] Product: `id`, `title`, `price`, `description`, `category`, `images[]`
- [x] Field audit: `created_at`, `created_by`, `created_by_id`, `updated_at`, `updated_by`, `updated_by_id`
- [x] Audit diisi dari claims JWT, bukan dari body request
- [x] User: `username` unik, `password_hash` bcrypt
- [x] RefreshToken: hash sha256, `expires_at`, `revoked_at`

## 3. Endpoint produk

- [x] `GET /api/products` — semua produk + query `search`, `category`, `limit`, `page`
- [x] `GET /api/products/:id` — detail, 404 JSON bila tidak ada
- [x] `POST /api/products` — validasi title, price, category, images minimal 1; 400 bila gagal
- [x] `PUT /api/products/:id` — edit, 404 bila tidak ada
- [x] `DELETE /api/products/:id` — hapus, 404 bila tidak ada
- [x] Guard authorization pada POST, PUT, DELETE

## 4. Endpoint auth

- [x] `POST /api/auth/register` — `username`, `password`, `password_confirmation`
- [x] 409 bila username bentrok
- [x] `POST /api/auth/login` — kembalikan `authentication_token` dan `refresh_token`
- [x] 401 bila kredensial salah
- [x] `POST /api/auth/refresh` — rotasi token, token lama dicabut
- [x] `POST /api/auth/logout` — cabut refresh token

## 5. Rate limit

- [x] Register dan login: 3 permintaan per 60 detik
- [x] Tulis produk: 1 permintaan per 5 detik
- [x] Counter di Redis dengan TTL, identifier user bila terautentikasi
- [x] Respons 429 + header `Retry-After`, `X-RateLimit-*`
- [x] Redis mati → request diteruskan, API tidak tumbang

## 6. Cache

- [x] `GET /api/products` cached 60 detik
- [x] `GET /api/products/:id` cached 300 detik
- [x] Invalidasi via `INCR products:ver`, tanpa `SCAN`/`KEYS`
- [x] Cache miss dan Redis mati → pass-through ke database

## 7. CORS

- [x] Echo `Origin` pemanggil + `Vary: Origin`
- [x] Izinkan `GET, POST, PUT, PATCH, DELETE, OPTIONS`
- [x] Izinkan header `Authorization`, `X-API-Token`, `Content-Type`
- [x] Preflight `OPTIONS` dijawab 204
- [x] `Access-Control-Expose-Headers` untuk header rate limit

## 8. Docker

- [x] `docker/Dockerfile` multi-stage, build statis, user non-root, healthcheck
- [x] `docker-compose.yml` — service api, postgres, redis, healthcheck + `depends_on`
- [x] Volume persisten untuk data PostgreSQL
- [x] `.env.example` dengan semua variabel
- [x] `docker compose up --build` dijalankan, ketiga container healthy
- [x] Seluruh endpoint diverifikasi manual via curl (health, list, filter, detail, 404, create, update, delete, register, login, refresh, logout, swagger)

## 9. Dokumentasi

- [x] Anotasi swaggo di seluruh handler
- [x] `make swag` menghasilkan `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`
- [x] Swagger UI tersedia di `/swagger/index.html`
- [x] README berisi cara jalan, tabel env, daftar endpoint, contoh curl, cara impor Postman
- [x] `make` targets: run, build, test, lint, fmt, swag, up, down, logs

## 10. Test

- [x] Unit test validasi payload dan pesan error
- [x] Unit test parsing query dan path parameter
- [x] Unit test ekstraksi token dan penolakan token JWT
- [x] Unit test pagination meta dan escaping
- [x] Test integrasi register, login, refresh, logout
- [x] Test integrasi CRUD produk, filter, pagination, 404
- [x] Test integrasi rate limit 429 dan `Retry-After`
- [x] Test integrasi cache hit dan invalidasi
- [x] Test integrasi CORS preflight
- [x] Isolasi rate limit test lewat `RATE_LIMIT_PREFIX` per test
- [x] Kontrak response diverifikasi: envelope `data` + `meta` pada list, `data` pada tunggal, `error.code` pada gagal
- [x] `alphanum` pada username diganti pola `^[a-zA-Z0-9._-]{3,30}$` agar menerima `jhon_doe` sesuai contoh di TODO
- [x] `go build ./...`, `go vet ./...`, `gofmt -l .` bersih
- [x] `go test ./...` hijau tanpa database
- [x] Test integrasi hijau dengan `TEST_DATABASE_URL` + `TEST_REDIS_ADDR` terhadap container docker
- [x] 81 test dijalankan, 0 gagal

## 11. Delivery

- [x] Repo git siap, `.gitignore` rapi
- [x] Commit Conventional Commits per tahap (4 commit)
- [ ] User mengirim URL remote repository
- [ ] Tambah `git remote` dan push (push dilakukan user)
