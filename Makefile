.PHONY: build test lint run setup venv test-py test-telegram docker-build docker-up docker-run-once backup restore

VENV := .venv
VENV_PY := $(VENV)/bin/python

build:
	go build -o bin/parser ./cmd/parser

setup: venv
	go mod download

venv:
	@test -x $(VENV_PY) || python3 -m venv $(VENV) --without-pip || python3 -m venv $(VENV)
	pip3 install --target "$$($(VENV_PY) -c 'import site; print(site.getsitepackages()[0])')" -r requirements.txt

test:
	go test ./...

test-py:
	PYTHONPATH=. $(if $(wildcard $(VENV_PY)),$(VENV_PY),python3) -m unittest discover -s sources/telegram -p 'test_*.py'

test-telegram: test-py
	go run ./cmd/parser -telegram-sidecar -telegram-dry-run -scan-once -output=quiet

lint:
	go vet ./...

run:
	go run ./cmd/parser

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-run-once:
	docker compose run --rm parser -scan-once

backup:
	./scripts/ops/backup-mongo.sh

restore:
	@test -n "$(DUMP)" || (echo "usage: make restore DUMP=backups/mongo-.../dump.gz" && exit 1)
	./scripts/ops/restore-mongo.sh "$(DUMP)"
