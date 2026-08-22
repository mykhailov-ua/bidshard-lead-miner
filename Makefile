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

.PHONY: build build-crm-bot crm-bot-smoke crm-caddy-up crm-caddy-down test lint fmt run setup venv test-py test-telegram docker-build docker-up docker-run-once backup restore proxy-check preflight-tgweb vps-preflight deploy-preflight ci ci-deploy-preflight tgweb-green-accept tgweb-discover-loop forum-live-check prod-source-smoke docker-headless-build bpf-release-gate bpf-leak-gate tgweb-bpf-leak-gate tgweb-seed tgweb-discover tgweb-prune tgweb-domains-prune tgweb-crawl tgweb-crawl-bpf tgweb-crawl-residential docker-tgweb-crawl vps-proxy-check vps-proxy-docker vps-proxy-down bpf-dev bpf-session-start bpf-session-stop

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

docker-headless-build:
	docker compose -f docker-compose.headless.yaml build

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

bpf-dev:
	bash scripts/dev/bpf_setup.sh

bpf-session-start: bpf-dev
	sudo bash scripts/dev/bpf_session.sh start

bpf-session-stop:
	sudo bash scripts/dev/bpf_session.sh stop
