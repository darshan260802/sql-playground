# syntax=docker/dockerfile:1

# ---- frontend: React + Vite
FROM node:22-bookworm-slim AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# ---- backend: Go API (static binary)
FROM golang:1.24-bookworm AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sqlpg ./cmd/sqlpg

# ---- runtime
FROM alpine:3.20
RUN adduser -D -H -u 10001 sqlpg
WORKDIR /app
COPY --from=api /out/sqlpg /app/sqlpg
COPY --from=web /web/dist /app/web
ENV SQLPG_LISTEN=:8080 \
    SQLPG_STATIC_DIR=/app/web
USER sqlpg
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=120s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health >/dev/null || exit 1
ENTRYPOINT ["/app/sqlpg"]
CMD ["serve"]
