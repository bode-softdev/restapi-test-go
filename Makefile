BINARY=bin/api
PKG=./...

.PHONY: help run build test test-integration lint fmt vet swag tidy up down logs clean docker-build docker-test

help:
	@echo "make run              jalankan API secara lokal"
	@echo "make build            compile biner ke $(BINARY)"
	@echo "make test             jalankan unit test"
	@echo "make test-integration jalankan test integrasi (butuh postgres)"
	@echo "make lint             gofmt check + go vet"
	@echo "make fmt              gofmt -w"
	@echo "make swag             generate ulang dokumentasi swagger"
	@echo "make tidy             go mod tidy"
	@echo "make up               docker compose up --build"
	@echo "make down             docker compose down"
	@echo "make logs             docker compose logs -f api"
	@echo "make clean            hapus artefak build"

run:
	go run ./cmd/api

build:
	go build -trimpath -o $(BINARY) ./cmd/api

test:
	go test -race -cover $(PKG)

test-integration:
	go test -race -count=1 -v ./test/...

lint:
	@gofmt -l . | tee /dev/stderr | (! read)
	go vet $(PKG)

fmt:
	gofmt -w .

vet:
	go vet $(PKG)

swag:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

tidy:
	go mod tidy

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api

clean:
	rm -rf bin

docker-build:
	docker compose build

docker-test:
	docker compose run --rm api sh -c "true"
