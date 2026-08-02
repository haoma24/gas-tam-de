# Gas Tam Đệ — local DX shortcuts (Sprint 0 / T9.2.3–T9.2.4)
# Usage: make help
# Windows without GNU Make: .\scripts\dev.ps1 help

COMPOSE := docker compose -f deploy/docker-compose.yml
GATEWAY_URL ?= http://127.0.0.1:8080
NATS_HEALTH_URL ?= http://127.0.0.1:8222/healthz

.PHONY: help nats-up nats-down nats-logs nats-init nats wait-nats \
	gateway auth catalog geo order inventory billing report \
	tidy test build compose-up compose-down health \
	flutter-get flutter-create flutter-web flutter-android flutter-ios flutter-devices

.DEFAULT_GOAL := help

help:
	@echo "Gas Tam De - make targets"
	@echo ""
	@echo "  Infra"
	@echo "    make nats-up      Start NATS JetStream (detached)"
	@echo "    make nats-down    Stop NATS container"
	@echo "    make nats-init    Bootstrap JetStream streams (cmd/nats-init)"
	@echo "    make nats         nats-up + wait + nats-init"
	@echo "    make nats-logs    Tail NATS logs"
	@echo "    make compose-up   Full stack docker compose --build"
	@echo "    make compose-down Stop full stack"
	@echo ""
	@echo "  Go services (host process, needs NATS for later sprints)"
	@echo "    make gateway | auth | catalog | geo | order | inventory | billing | report"
	@echo "    make tidy         go mod tidy"
	@echo "    make test         go test ./..."
	@echo "    make build        go build ./services/..."
	@echo "    make health       GET gateway /healthz + /v1/hello"
	@echo ""
	@echo "  Flutter (apps/mobile) — Web + Android + iOS CTA shell (T9.2.4)"
	@echo "    make flutter-get      flutter pub get"
	@echo "    make flutter-create   flutter create . --platforms=web,android,ios"
	@echo "    make flutter-web      flutter run -d chrome"
	@echo "    make flutter-android  flutter run -d android"
	@echo "    make flutter-ios      flutter run -d ios"
	@echo "    make flutter-devices  flutter devices"
	@echo ""
	@echo "  Windows tip: .\\scripts\\dev.ps1 help  (same commands without GNU Make)"

# --- Infra / NATS ---

nats-up:
	$(COMPOSE) up nats -d

nats-down:
	$(COMPOSE) stop nats

nats-logs:
	$(COMPOSE) logs -f nats

wait-nats:
	@echo "Waiting for NATS health at $(NATS_HEALTH_URL) ..."
	@i=0; \
	while [ $$i -lt 30 ]; do \
		if curl -sf "$(NATS_HEALTH_URL)" >/dev/null 2>&1; then \
			echo "NATS healthy"; \
			exit 0; \
		fi; \
		i=$$((i+1)); \
		sleep 1; \
	done; \
	echo "NATS did not become healthy in time"; \
	exit 1

nats-init:
	go run ./cmd/nats-init

nats: nats-up wait-nats nats-init

compose-up:
	$(COMPOSE) up --build

compose-down:
	$(COMPOSE) down

# --- Go services ---

gateway:
	go run ./services/api-gateway

auth:
	go run ./services/auth-service

catalog:
	go run ./services/catalog-service

geo:
	go run ./services/geo-service

order:
	go run ./services/order-service

inventory:
	go run ./services/inventory-service

billing:
	go run ./services/billing-service

report:
	go run ./services/report-service

tidy:
	go mod tidy

test:
	go test ./...

build:
	go build ./services/...

health:
	curl -sf "$(GATEWAY_URL)/healthz"
	@echo
	curl -sf "$(GATEWAY_URL)/v1/hello"
	@echo

# --- Flutter (T9.2.4 multi-platform CTA shell) ---

flutter-get:
	cd apps/mobile && flutter pub get

flutter-create:
	cd apps/mobile && flutter create . --project-name gas_tam_de --org vn.gastamde --platforms=web,android,ios

flutter-web:
	cd apps/mobile && flutter run -d chrome

flutter-android:
	cd apps/mobile && flutter run -d android

flutter-ios:
	cd apps/mobile && flutter run -d ios

flutter-devices:
	cd apps/mobile && flutter devices
