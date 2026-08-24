# ── Frontend build stage ───────────────────────────────────────────────
FROM node:22-alpine AS frontend

WORKDIR /src/internal/web/frontend

COPY internal/web/frontend/package.json internal/web/frontend/package-lock.json ./
RUN npm ci

COPY internal/web/frontend/ ./
RUN npm run build

# ── Go build stage ─────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy source (excluding frontend node_modules via .dockerignore).
COPY . .

# Copy the built frontend dist into the expected embed location.
COPY --from=frontend /src/internal/web/frontend/dist ./internal/web/frontend/dist

# Build the binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /usr/local/bin/proxy .

# ── Runtime stage ──────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /usr/local/bin/proxy /usr/local/bin/proxy

# Default data directory for the SQLite database.
RUN mkdir -p /data
ENV DSN=/data/proxy.db

EXPOSE 8080

ENTRYPOINT ["proxy"]
