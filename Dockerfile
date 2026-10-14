# ============================================================
# Dockerfile multi-stage untuk SPLK Backend
# Membangun dua binary: api dan seed
# ============================================================

# --- Tahap build ---
FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Salin file dependency terlebih dahulu agar layer cache efektif
COPY go.mod go.sum ./
RUN go mod download

# Salin seluruh kode sumber
COPY . .

# Build dua binary dengan CGO dinonaktifkan
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/bin/seed ./cmd/seed

# --- Tahap runtime ---
FROM alpine:3.20

# Paket yang dibutuhkan runtime
RUN apk add --no-cache ca-certificates tzdata

# Atur zona waktu
ENV TZ=Asia/Makassar

# Buat user non-root
RUN addgroup -S splk && adduser -S splk -G splk

WORKDIR /app

# Salin binary dari tahap build
COPY --from=builder /build/bin/api ./api
COPY --from=builder /build/bin/seed ./seed

# Buat direktori uploads
RUN mkdir -p /app/uploads && chown -R splk:splk /app

# Jalankan sebagai user non-root
USER splk

EXPOSE 8080

CMD ["./api"]
