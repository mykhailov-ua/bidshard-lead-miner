# syntax=docker/dockerfile:1
#
# Multi-stage image: Go parser + Python Telethon sidecar deps.
#
# Usage:
#   docker compose build
#   docker compose run --rm parser scan
#   docker compose run --rm -it parser telegram login --qr
#
# Default CMD: parser run (override with scan, telegram, config check, ...).
# Entrypoint drops to UID 10001 and chowns data/runtime + data/export for bind mounts.

FROM golang:1.25-alpine AS go-builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /parser ./cmd/parser && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /crm-bot ./cmd/crm-bot

FROM python:3.12-alpine AS py-builder

WORKDIR /deps
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

FROM alpine:3.20

RUN apk add --no-cache ca-certificates python3 su-exec && \
    adduser -D -u 10001 -h /app parser

WORKDIR /app

COPY --from=py-builder /install /usr
COPY --from=go-builder /parser /usr/local/bin/parser
COPY --from=go-builder /crm-bot /usr/local/bin/crm-bot

COPY --chown=parser:parser data/ /app/data/
COPY --chown=parser:parser config/ /app/config/
COPY --chown=parser:parser sources/ /app/sources/
COPY docker-entrypoint.sh /docker-entrypoint.sh

ENV KEYWORDS_JSON_PATH=/app/data/keywords.json \
    KEYWORDS_GRAY_JSON_PATH=/app/data/keywords-gray.json \
    TELEGRAM_CONFIG_PATH=/app/config/sources.telegram.yaml \
    SUPPLY_SEED_PATH=/app/data/seeds/domains.csv \
    FORUM_SEED_PATH=/app/data/seeds/forum_threads.csv \
    LANDER_SEED_PATH=/app/data/seeds/lander_urls.csv \
    PYTHONPATH=/app

RUN mkdir -p /app/data/export /app/data/runtime && \
    chown -R parser:parser /app/data && \
    chmod +x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

# Continuous scan loop (override with `scan` for one-shot / cron)
CMD ["run"]
