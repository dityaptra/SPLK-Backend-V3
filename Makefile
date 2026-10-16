# ============================================================
# Makefile — SPLK Backend
# ============================================================

-include .env
export

# Variabel
BINARY_API  = bin/api
BINARY_SEED = bin/seed
MIGRATE_DB  ?= $(if $(DATABASE_URL),$(DATABASE_URL),postgres://$(if $(DB_USER),$(DB_USER),splk):$(if $(DB_PASSWORD),$(DB_PASSWORD),ganti_password_ini)@$(if $(DB_HOST),$(DB_HOST),localhost):$(if $(DB_PORT),$(DB_PORT),5432)/$(if $(DB_NAME),$(DB_NAME),splk_db)?sslmode=$(if $(DB_SSLMODE),$(DB_SSLMODE),disable))

# --- Pengembangan ---

.PHONY: run
run: ## Jalankan server API
	go run ./cmd/api

.PHONY: dev
dev: ## Jalankan server dengan hot reload (air)
	air

.PHONY: build
build: ## Build binary API dan Seed
	go build -o $(BINARY_API) ./cmd/api
	go build -o $(BINARY_SEED) ./cmd/seed

.PHONY: test
test: ## Jalankan semua test
	go test ./... -v

.PHONY: fmt
fmt: ## Format kode Go
	gofmt -w .

.PHONY: vet
vet: ## Jalankan go vet
	go vet ./...

.PHONY: tidy
tidy: ## Rapikan go.mod dan go.sum
	go mod tidy

# --- Migrasi ---

.PHONY: migrate-up
migrate-up: ## Jalankan migrasi database ke versi terbaru
	migrate -path migrations -database "$(MIGRATE_DB)" up

.PHONY: migrate-down
migrate-down: ## Rollback migrasi satu langkah
	migrate -path migrations -database "$(MIGRATE_DB)" down 1

.PHONY: migrate-create
migrate-create: ## Buat file migrasi baru (make migrate-create name=nama_migrasi)
	migrate create -ext sql -dir migrations -seq $(name)

# --- Seed ---

.PHONY: seed
seed: ## Jalankan seeder admin
	go run ./cmd/seed

# --- Docker ---

.PHONY: docker-up
docker-up: ## Build dan jalankan container
	docker compose up --build -d

.PHONY: docker-down
docker-down: ## Hentikan container
	docker compose down

.PHONY: docker-logs
docker-logs: ## Lihat log container
	docker compose logs -f

# --- Swagger ---

.PHONY: swag
swag: ## Generate dokumentasi Swagger
	swag init -g cmd/api/main.go -o docs

.PHONY: help
help: ## Tampilkan bantuan
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
