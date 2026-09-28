# REST API Product Management

REST API manajemen produk dengan autentikasi JWT, cache, dan rate limit. Dibangun dengan Go, PostgreSQL, dan Redis, serta dapat dijalankan lewat Docker.

## Fitur

- CRUD produk dengan filter `search`, `category`, dan pagination `page` / `limit`
- Register, login, refresh token (rotasi), dan logout
- Autentikasi `Authorization: Bearer <authentication_token>`
- Rate limit per endpoint di atas Redis
- Cache respons produk di Redis dengan invalidasi berbasis versi
- CORS terbuka agar API bisa diakses domain web lain
- Dokumentasi Swagger 2.0 di `/swagger/index.html`
- Error JSON konsisten untuk seluruh status

## Teknologi

| Bagian             | /library                |
| ------------------ | ----------------------- |
| Bahasa             | Go 1.27                 |
| HTTP               | Gin                     |
| ORM                | GORM                    |
| Database           | PostgreSQL 17           |
| Cache + rate limit | Redis 7 (go-redis)      |
| Auth               | golang-jwt/v5 + bcrypt  |
| Docs               | swaggo (Swagger 2.0)    |
| Runtime            | Docker + Docker Compose |

## Menjalankan dengan Docker

```bash
cp .env.example .env
# isi JWT_SECRET dengan nilai acak, minimal 32 karakter
openssl rand -base64 48

docker compose up --build -d
curl http://localhost:8080/health
```

Swagger UI: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

Matikan: `docker compose down`. Hapus juga volume database: `docker compose down -v`.

## Menjalankan secara lokal

Butuh PostgreSQL dan Redis yang sudah jalan di host.

```bash
cp .env.example .env
# batasi DATABASE_URL dan REDIS_ADDR ke localhost
go run ./cmd/api
```

Saat start, aplikasi melakukan migrasi otomatis dan mengisi data contoh bila tabel masih kosong:

- user: `jhon_doe` / `supersecret`
- 5 produk contoh pada kategori Clothes, Electronics, Food

## Konfigurasi

Semua variabel dibaca dari environment atau file `.env`.

| Variabel              | Default                                                               | Keterangan                                  |
| --------------------- | --------------------------------------------------------------------- | ------------------------------------------- |
| `APP_PORT`          | `8080`                                                              | Port HTTP                                   |
| `GIN_MODE`          | `release`                                                           | Mode Gin:`debug` / `release` / `test` |
| `DATABASE_URL`      | `postgres://restapi:restapi@localhost:5432/restapi?sslmode=disable` | DSN PostgreSQL                              |
| `REDIS_ADDR`        | `localhost:6379`                                                    | Alamat Redis                                |
| `REDIS_PASSWORD`    | kosong                                                                | Password Redis                              |
| `REDIS_DB`          | `0`                                                                 | Indeks database Redis                       |
| `JWT_SECRET`        | wajib                                                                 | Minimal 32 karakter, HMAC HS256             |
| `ACCESS_TOKEN_TTL`  | `15m`                                                               | Masa berlaku access token                   |
| `REFRESH_TOKEN_TTL` | `720h`                                                              | Masa berlaku refresh token                  |
| `BCRYPT_COST`       | `10`                                                                | Cost bcrypt, 4-31                           |
| `CACHE_TTL_LIST` | `60s` | TTL cache daftar produk |
| `CACHE_TTL_ITEM` | `300s` | TTL cache detail produk |
| `CONNECT_ATTEMPTS` | `30` | Berapa kali mencoba koneksi database saat start |
| `CONNECT_BACKOFF` | `1s` | Jeda awal antar percobaan, digandakan tiap gagal |
| `RATE_LIMIT_PREFIX` | `rl` | Prefix key rate limit di Redis |

Bila Redis mati, API tetap berjalan tanpa cache dan tanpa rate limit.

Port default `APP_PORT` adalah `8080` dan `POSTGRES_PORT` adalah `5432`. Ubah keduanya di `.env` bila sudah dipakai proses lain di mesin ini.

## Endpoint

| Method     | Path                   | Auth          | Rate limit   | Keterangan                   |
| ---------- | ---------------------- | ------------- | ------------ | ---------------------------- |
| `GET`    | `/health`            | -             | -            | Liveness probe               |
| `POST`   | `/api/auth/register` | -             | 3 / 60 detik | Buat akun                    |
| `POST`   | `/api/auth/login`    | -             | 3 / 60 detik | Ambil token                  |
| `POST`   | `/api/auth/refresh`  | refresh token | 3 / 60 detik | Rotasi token                 |
| `POST`   | `/api/auth/logout`   | refresh token | 3 / 60 detik | Cabut refresh token          |
| `GET`    | `/api/products`      | -             | -            | Daftar + filter + pagination |
| `GET`    | `/api/products/:id`  | -             | -            | Detail produk                |
| `POST`   | `/api/products`      | Bearer        | 1 / 5 detik  | Tambah produk                |
| `PUT`    | `/api/products/:id`  | Bearer        | 1 / 5 detik  | Edit produk                  |
| `DELETE` | `/api/products/:id`  | Bearer        | 1 / 5 detik  | Hapus produk                 |

Rate limit dihitung per user bila request terautentikasi, selain itu per alamat IP. Saat limit terlampaui, respons `429` beserta header `Retry-After`.

## Bentuk respons

Berhasil list:

```json
{
  "data": [ { "id": 1, "title": "Awesome T-Shirt", "price": 99.99 } ],
  "meta": { "total": 5, "page": 1, "limit": 10, "total_pages": 1, "has_next": false, "has_previous": false }
}
```

Berhasil tunggal: `{ "data": { ... } }`. Hapus: `{ "data": null, "message": "produk berhasil dihapus" }`.

Gagal:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "data produk tidak valid",
    "details": { "title": "wajib diisi" }
  }
}
```

| Status | `code`                    | Arti                            |
| ------ | --------------------------- | ------------------------------- |
| 400    | `VALIDATION_ERROR`        | Data tidak memenuhi aturan      |
| 400    | `INVALID_QUERY_PARAMETER` | Query atau path parameter salah |
| 401    | `UNAUTHORIZED`            | Token tidak dikirim             |
| 401    | `TOKEN_INVALID`           | Token tidak bisa diverifikasi   |
| 401    | `TOKEN_EXPIRED`           | Access token kedaluwarsa        |
| 404    | `NOT_FOUND`               | Produk atau endpoint tidak ada  |
| 409    | `CONFLICT`                | Username sudah dipakai          |
| 429    | `RATE_LIMITED`            | Terlalu banyak permintaan       |
| 500    | `INTERNAL_ERROR`          | Kesalahan server                |

## Contoh pemakaian

```bash
BASE=http://localhost:8080

curl -s -X POST $BASE/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"budi","password":"supersecret","password_confirmation":"supersecret"}'

TOKEN=$(curl -s -X POST $BASE/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"budi","password":"supersecret"}' | jq -r .authentication_token)

curl -s "$BASE/api/products?search=shirt&category=Clothes&limit=10&page=1"

curl -s -X POST $BASE/api/products \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Kaos Katun","price":89000,"description":"Katun combed 30s","category":"Clothes","images":["https://placeimg.com/640/480/any"]}'

curl -s -X PUT $BASE/api/products/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"price":95000}'

curl -s -X DELETE $BASE/api/products/1 -H "Authorization: Bearer $TOKEN"
```

## Develop

```bash
make run                # jalankan lokal
make test               # unit test
make test-integration   # test integrasi, butuh postgres + redis
make lint               # gofmt check + go vet
make swag               # regenerate swagger
make up / make down     # docker compose
```

Test integrasi memakai database terpisah dan otomatis dilewati bila env tidak diisi:

```bash
docker compose exec postgres createdb -U restapi restapi_test
docker exec restapi-redis redis-cli FLUSHDB

$env:TEST_DATABASE_URL="postgres://restapi:restapi@localhost:5432/restapi_test?sslmode=disable"
$env:TEST_REDIS_ADDR="localhost:6379"
make test-integration
```

Gunakan `POSTGRES_PORT` dari `.env` bila port host Postgres bukan `5432`. Setiap test memakai prefix rate limit sendiri lewat `RATE_LIMIT_PREFIX`, jadi test tidak saling memengaruhi.

## Impor ke Postman

Swagger 2.0 tersedia di [http://localhost:8080/swagger/doc.json](http://localhost:8080/swagger/doc.json). Di Postman: **Import → Link**, tempel URL tersebut.

## Struktur proyek

```
cmd/api/          titik masuk aplikasi
internal/
  cache/          cache Redis + invalidasi berbasis versi
  config/         pemuatan konfigurasi
  database/       koneksi, migrasi, data awal
  handler/        handler HTTP + anotasi swagger
  middleware/     auth, rate limit, CORS, logger
  model/          struct GORM
  repository/     query database
  response/       kontrak JSON dan helper
  router/         registrasi route
  service/        logika bisnis
test/             test integrasi end-to-end
docs/             hasil generate swagger
docker/           Dockerfile
```

## Lisensi

MIT
