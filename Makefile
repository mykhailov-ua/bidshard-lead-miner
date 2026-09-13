# BidShard lead-intent-processor - common dev and ops targets.
#
# Quick start:
#   cp .env.example .env && docker compose up -d mongo
#   make build && make test
#
# tgweb:
#   make tgweb-seed && make preflight-tgweb && make tgweb-crawl
#   make docker-tgweb-crawl DOMAINS=buylink.pro,topxpartners.com
#
# BPF dev (Linux, root):
#   make bpf-dev && sudo make bpf-session-start
#
# Docs: README.md, docs/OPS.md, docs/CREDENTIALS.md, docs/DEPLOY.md

.PHONY: build build-crm-bot crm-bot-smoke crm-caddy-up crm-caddy-down test lint fmt run setup venv test-py test-telegram docker-build docker-up docker-run-once backup restore proxy-check preflight-tgweb vps-preflight vps-deploy vps-deploy-p0 vps-sync vps-sync-proxy vps-sync-telegram-secrets vps-sync-telethon-session vps-sync-session-pool vps-install-telegram-cron session-pool-link vps-telegram-login vps-telegram-login-qr vps-history-export vps-reddit-offline-archive vps-buyer-discover reddit-offline-archive vps-export lip-install-shell lead-logs deploy-preflight ci ci-deploy-preflight tgweb-green-accept tgweb-discover-loop forum-live-check prod-source-smoke acceptance-soak warm-path-status docker-headless-build bpf-release-gate bpf-leak-gate tgweb-bpf-leak-gate tgweb-seed tgweb-discover tgweb-prune tgweb-domains-prune tgweb-crawl tgweb-crawl-bpf tgweb-crawl-residential docker-tgweb-crawl vps-proxy-check vps-proxy-docker vps-proxy-down bpf-dev bpf-session-start bpf-session-stop buyer-discover

VENV := .venv
VENV_PY := $(VENV)/bin/python
BIN_DIR := bin

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	go build -ldflags "-X github.com/bidshard/parser/cmd/parser.Version=dev" -o $(BIN_DIR)/parser ./cmd/parser

build-crm-bot: $(BIN_DIR)
	go build -ldflags "-X github.com/bidshard/parser/cmd/crm-bot.Version=dev" -o $(BIN_DIR)/crm-bot ./cmd/crm-bot

crm-bot-smoke:
	bash scripts/dev/crm_bot_smoke.sh

geo-funnel-report:
	bash scripts/dev/geo_funnel_report.sh

entity-heat-report:
	bash scripts/dev/entity_heat_report.sh

crm-caddy-up:
	bash scripts/dev/crm_caddy_up.sh

crm-caddy-down:
	docker compose -f docker-compose.crm-edge.yaml down

setup: venv
	go mod download

venv:
	@test -x $(VENV_PY) || python3 -m venv $(VENV) --without-pip || python3 -m venv $(VENV)
	pip3 install --target "$$($(VENV_PY) -c 'import site; print(site.getsitepackages()[0])')" -r requirements.txt -r requirements-headless.txt -r requirements-dev.txt

test:
	go test ./...

test-integration:
	go test -tags=integration ./internal/sink/... ./internal/crm/store/...

test-py:
	PYTHONPATH=. $(if $(wildcard $(VENV_PY)),$(VENV_PY),python3) -m unittest discover -s sources/telegram -p 'test_*.py'

test-telegram: test-py
	go run ./cmd/parser telegram --dry-run --output=quiet

lint:
	go vet ./...
	$(if $(wildcard $(VENV_PY)),$(VENV_PY),python3) -m ruff check sources scripts
	$(if $(wildcard $(VENV_PY)),$(VENV_PY),python3) -m pyright sources scripts

fmt:
	gofmt -w $$(git ls-files '*.go')
	goimports -w $$(git ls-files '*.go')
	@if command -v ruff >/dev/null 2>&1; then ruff format $$(git ls-files '*.py'); fi

run:
	go run ./cmd/parser run

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-run-once:
	docker compose run --rm parser scan

backup:
	./scripts/ops/backup-mongo.sh

restore:
	@test -n "$(DUMP)" || (echo "usage: make restore DUMP=backups/mongo-.../dump.gz" && exit 1)
	./scripts/ops/restore-mongo.sh "$(DUMP)"

proxy-check:
	./scripts/proxy/check-proxy.sh

preflight-tgweb:
	bash ./scripts/proxy/preflight-tgweb.sh

vps-preflight: build
	bash ./scripts/proxy/vps-preflight.sh

# VPS rsync + docker compose up (config: config/env/.env.vps-deploy.local). See scripts/ops/lip.
vps-deploy:
	bash ./scripts/ops/vps-deploy.sh

# P0: BidShard ICP env + parser + crm-bot + cron telegram scrape (see SHARDING_SESSION.md)
p0-secrets-check:
	bash ./scripts/ops/p0-secrets-check.sh

p0-secrets-check-vps:
	bash ./scripts/ops/p0-secrets-check.sh --vps

vps-deploy-p0:
	bash ./scripts/ops/vps-deploy-p0.sh

vps-status:
	bash ./scripts/ops/vps-status.sh

vps-p0-telegram-soak:
	bash ./scripts/ops/vps-p0-telegram-soak.sh

vps-realtime-soak:
	bash ./scripts/ops/vps-realtime-soak.sh

vps-telegram-realtime-soak:
	bash ./scripts/ops/vps-telegram-realtime-soak.sh

vps-history-export:
	bash ./scripts/ops/vps-history-export.sh $(ARGS)

reddit-offline-archive:
	bash ./scripts/ops/reddit-offline-archive.sh $(ARGS)

vps-reddit-offline-archive:
	bash ./scripts/ops/vps-reddit-offline-archive.sh $(ARGS)

vps-buyer-discover:
	bash ./scripts/ops/vps-buyer-discover.sh $(ARGS)

vps-sync-telegram-secrets:
	bash ./scripts/ops/vps-sync-telegram-secrets.sh

vps-sync-discord-secrets:
	bash ./scripts/ops/vps-sync-discord-secrets.sh

discord-pool-link:
	bash ./scripts/ops/discord-pool-link.sh

discord-discover:
	bash ./scripts/ops/discord-discover.sh

vps-sync-telethon-session:
	bash ./scripts/ops/vps-sync-telethon-session.sh

session-pool-link:
	bash ./scripts/ops/session-pool-link.sh

vps-sync-session-pool:
	bash ./scripts/ops/vps-sync-session-pool.sh

vps-install-telegram-cron:
	bash ./scripts/ops/vps-install-telegram-cron.sh

vps-telegram-login:
	bash ./scripts/ops/vps-telegram-login.sh phone

vps-telegram-login-qr:
	bash ./scripts/ops/vps-telegram-login.sh qr

setup-telegram-alert-group:
	bash ./scripts/ops/setup-telegram-alert-group.sh

setup-telegram-alert-group-vps:
	bash ./scripts/ops/setup-telegram-alert-group.sh --vps

vps-sync:
	VPS_SYNC_ONLY=1 bash ./scripts/ops/vps-deploy.sh

# Upload config/env/proxy.list (from proxy.txt) to VPS; never commit proxy.list.
vps-sync-proxy:
	bash ./scripts/ops/sync-proxy-to-vps.sh

vps-export:
	bash ./scripts/ops/lip export

lip-install-shell:
	bash ./scripts/ops/lip install-shell

lead-logs:
	bash ./scripts/ops/lead-logs

# Pre-deploy: vps-preflight + tgweb crawl under eBPF leak probe (Linux + sudo). See scripts/proxy/deploy-preflight.sh
deploy-preflight: build
	bash ./scripts/proxy/deploy-preflight.sh

# CI gates (Go test, slop, Python, BPF fixture leak-gate on Linux).
ci:
	bash ./scripts/ci/run.sh

# CI deploy preflight (no proxy, host tgweb crawl). See scripts/ci/deploy-preflight.sh
ci-deploy-preflight: build
	bash ./scripts/ci/deploy-preflight.sh

forum-live-check: build
	bash ./scripts/sources/forum-live-check.sh

prod-source-smoke: build
	bash ./scripts/sources/prod-source-smoke.sh

# Epic J: jq gates on JSONL export (pending %, lander junk, CSS contacts, telegram High pain).
acceptance-soak:
	bash ./scripts/ops/acceptance-soak.sh

icp-soak-report:
	bash ./scripts/ops/icp-soak-report.sh

icp-soak-report-vps:
	bash ./scripts/ops/icp-soak-report.sh --vps

# Warm-path pending/DLQ snapshot (requires mongosh + MONGO_URI).
warm-path-status:
	bash ./scripts/ops/warm-path-status.sh

docker-headless-build:
	docker compose -f docker-compose.headless.yaml build

headless-crawl-cron:
	bash scripts/ops/headless-crawl-cron.sh

bpf-release-gate:
	@test -n "$(SESSION)" || (echo "usage: make bpf-release-gate SESSION=var/bpf-session/<ts>" && exit 1)
	bash ./scripts/lib/bpf_gate.sh "$(SESSION)"

bpf-leak-gate:
	@test -n "$(SESSION)" || (echo "usage: make bpf-leak-gate SESSION=var/bpf-session/<ts>" && exit 1)
	bash ./scripts/lib/bpf_leak_gate.sh "$(SESSION)"

# Pre-deploy: tgweb crawl + eBPF session + strict leak-gate (Linux + sudo). See scripts/tgweb/bpf-leak-preflight.sh
tgweb-bpf-leak-gate:
	bash ./scripts/tgweb/bpf-leak-preflight.sh

tgweb-green-accept: build
	bash ./scripts/tgweb/green-accept.sh

tgweb-discover-loop:
	bash ./scripts/tgweb/discover-loop.sh

tgweb-seed:
	bash ./scripts/tgweb/seed-registry.sh

tgweb-discover:
	docker compose -f docker-compose.tgweb.yaml --profile tgweb run --rm tgweb-discover

# SERP harvest (discover.icp.json) + CF crawl via residential proxy (forum,tgweb,serp).
buyer-discover:
	bash ./scripts/ops/buyer-discover.sh

vps-install-buyer-discover-cron:
	bash ./scripts/ops/vps-install-buyer-discover-cron.sh

tgweb-prune:
	docker compose -f docker-compose.tgweb.yaml --profile tgweb run --rm tgweb-prune

tgweb-domains-prune: build
	./bin/parser telegram domains prune

tgweb-crawl:
	bash ./scripts/tgweb/crawl.sh

tgweb-crawl-bpf:
	PARSER_BPF_BASELINE=1 bash ./scripts/tgweb/crawl.sh

tgweb-crawl-residential:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; \
		if [ -z "$${PARSER_PROXY_LIST//[[:space:]]/}" ]; then \
			echo "tgweb-crawl-residential: set PARSER_PROXY_LIST in .env (see config/env/.env.residential.example)"; \
			exit 1; \
		fi'
	bash ./scripts/tgweb/crawl.sh

docker-tgweb-crawl:
	docker compose -f docker-compose.tgweb.yaml --profile tgweb run --rm tgweb-crawl $(if $(DOMAINS),--domains $(DOMAINS),)

vps-proxy-check: proxy-check

vps-proxy-docker:
	./scripts/vps-proxy/setup-docker-proxy.sh

vps-proxy-down:
	docker compose -f scripts/vps-proxy/docker-compose.proxy.yaml down

# Home ISP egress: run proxy + tunnel on your PC, then vps-apply-home-proxy (see scripts/home-egress/README.md)
home-egress-proxy-up:
	chmod +x scripts/home-egress/*.sh scripts/ops/vps-check-home-tunnel.sh
	./scripts/home-egress/setup-home-proxy.sh

home-egress-tunnel:
	./scripts/home-egress/tunnel-to-vps.sh

home-egress-systemd:
	chmod +x scripts/home-egress/install-user-systemd.sh
	./scripts/home-egress/install-user-systemd.sh

home-egress-env:
	./scripts/home-egress/print-vps-env.sh

vps-check-home-tunnel:
	bash ./scripts/ops/vps-check-home-tunnel.sh

vps-apply-home-proxy:
	bash ./scripts/ops/vps-apply-home-proxy.sh

vps-save-residential-proxy:
	bash ./scripts/ops/vps-save-residential-proxy.sh

vps-install-proxy-failover:
	bash ./scripts/ops/install-proxy-egress-failover.sh

bpf-dev:
	bash scripts/dev/bpf_setup.sh

bpf-session-start: bpf-dev
	sudo bash scripts/dev/bpf_session.sh start

bpf-session-stop:
	sudo bash scripts/dev/bpf_session.sh stop
