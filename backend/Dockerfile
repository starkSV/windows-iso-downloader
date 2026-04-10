# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o msdl-backend main.go

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

# Security: run as non-root
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy binary and optional static public folder
COPY --from=builder /app/msdl-backend .
COPY --from=builder /app/public ./public

RUN chown -R app:app /app
USER app

EXPOSE 3002

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:3002/health || exit 1

CMD ["./msdl-backend"]
