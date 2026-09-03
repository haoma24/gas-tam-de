# Gas Tam Đệ — local DX shortcuts (Sprint 0 / T9.2.3–T9.2.4)
# Usage: make help
# Windows without GNU Make: .\scripts\dev.ps1 help

COMPOSE_FILE := deploy/docker-compose.yml
# Local host-port mappings (8080/4222/8090). VPS/Coolify must NOT load this file —
# publishing api-gateway:8080 breaks Traefik Stage 8 (Unreachable).
COMPOSE_LOCAL := $(wildcard deploy/docker-compose.local.yml)
COMPOSE_ENV := $(wildcard deploy/.env)
COMPOSE := docker compose -f $(COMPOSE_FILE) $(if $(COMPOSE_LOCAL),-f $(COMPOSE_LOCAL),) $(if $(COMPOSE_ENV),--env-file $(COMPOSE_ENV),)
GATEWAY_URL ?= http://127.0.0.1:8080
WEB_URL ?= http://127.0.0.1:$(or $(WEB_PORT),8090)
PROXY_NETWORK ?= tensorship-net
NATS_HEALTH_URL ?= http://127.0.0.1:8222/healthz
SWAG_VERSION := v1.16.6
SWAG_DIRS := services/api-gateway,services/auth-service,services/catalog-service,services/geo-service,services/order-service,services/inventory-service,services/billing-service,services/report-service,pkg/httpx
GOOGLE_WEB_CLIENT_ID ?=
GOOGLE_IOS_CLIENT_ID ?=

.PHONY: help nats-up nats-down nats-logs nats-init nats wait-nats \
	gateway auth catalog geo order inventory billing report \
	tidy test build swagger swagger-check compose-up compose-down compose-ps compose-logs \
	web-up web-logs web-health health stack-health doctor check-env-yaml \
	vps-net-check vps-net-fix vps-up vps-api-diagnose ensure-proxy-net \
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
	@echo "    make compose-up   Full stack docker compose --build (waits for healthy)"
	@echo "    make compose-down Stop full stack"
	@echo "    make compose-ps   Status of every container (incl. health)"
	@echo "    make compose-logs Tail logs of ALL services (not just NATS)"
	@echo "    make doctor       Why is a container unhealthy? (status + probe + logs)"
	@echo "    make check-env-yaml  Ensure .env.example is safe for Cursor Cloud override YAML"
	@echo "    make vps-net-check   Which containers are missing from the platform proxy net?"
	@echo "    make vps-net-fix     Attach those containers (deploy said cause=NotOnNet)"
	@echo "    make vps-up          Pull GHCR :stag images + up --no-build (VPS manual)"
	@echo "    make vps-api-diagnose  Test web → api-gateway from inside stack"
	@echo ""
	@echo "  Website (Flutter Web + nginx in Docker, port 8090)"
	@echo "    make web-up       Build + start nats, auth, gateway, web (OTP-ready)"
	@echo "    make web-logs     Tail nginx access/error logs"
	@echo "    make web-health   GET web /web-healthz + /gateway-healthz + /v1/hello"
	@echo "    make stack-health compose-ps + gateway health + web health"
	@echo ""
	@echo "  Go services (host process, needs NATS for later sprints)"
	@echo "    make gateway | auth | catalog | geo | order | inventory | billing | report"
	@echo "    make tidy         go mod tidy"
	@echo "    make test         go test ./..."
	@echo "    make build        go build ./services/..."
	@echo "    make swagger      Regenerate gateway Swagger docs from service annotations"
	@echo "    make swagger-check  Fail when generated Swagger docs are stale"
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

nats-up: ensure-proxy-net
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

compose-up: ensure-proxy-net
	$(COMPOSE) up --build -d --wait
	@$(MAKE) --no-print-directory compose-ps

compose-down:
	$(COMPOSE) down

# Cursor Cloud Stage 5 generates docker-compose.override.yml from .env.example
# with unquoted YAML values — catch ": " breakages before deploy.
check-env-yaml:
	@./scripts/check-env-yaml-safe.sh

# Deploy reported "HEALTHCHECK FAILED cause=NotOnNet" — the platform failed to
# attach a container to its Traefik network. Override with PROXY_NETWORK=<name>.
vps-net-check:
	@./scripts/vps-net-check.sh

vps-net-fix:
	@./scripts/vps-net-check.sh --fix

# VPS manual deploy when the platform runs `compose build` (should no-op after
# build contexts moved to docker-compose.local.yml). Pulls pinned :stag images.
VPS_COMPOSE_PROJECT ?= ts-tamde-stag
ensure-proxy-net:
	@docker network inspect "$(PROXY_NETWORK)" >/dev/null 2>&1 || docker network create "$(PROXY_NETWORK)"

vps-up:
	COMPOSE_PROJECT_NAME=$(VPS_COMPOSE_PROJECT) PROXY_NETWORK=$(PROXY_NETWORK) ./scripts/vps-compose-up.sh

vps-api-diagnose:
	COMPOSE_PROJECT_NAME=$(VPS_COMPOSE_PROJECT) PROXY_NETWORK=$(PROXY_NETWORK) ./scripts/vps-api-diagnose.sh

compose-ps:
	$(COMPOSE) ps -a

compose-logs:
	$(COMPOSE) logs -f --tail=100

# Prints, for every container that is not healthy, the last health-probe output
# and the tail of its logs — the two things needed to explain "is unhealthy".
doctor:
	@echo "=== containers ==="
	@$(COMPOSE) ps -a
	@echo
	@for c in $$($(COMPOSE) ps -aq); do \
		name=$$(docker inspect --format '{{.Name}}' $$c | sed 's|^/||'); \
		state=$$(docker inspect --format '{{.State.Status}}' $$c); \
		health=$$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' $$c); \
		if [ "$$health" = "healthy" ] && [ "$$state" = "running" ]; then continue; fi; \
		echo "=== $$name (state=$$state health=$$health) ==="; \
		docker inspect --format '{{if .State.Health}}last probe: {{range $$i, $$l := .State.Health.Log}}{{if eq $$i 0}}{{end}}{{end}}{{(index .State.Health.Log 0).Output}}{{end}}' $$c 2>/dev/null || true; \
		echo "--- last 40 log lines ---"; \
		docker logs --tail=40 $$c 2>&1 || true; \
		echo; \
	done
	@echo "All healthy containers omitted. Nothing above means the stack is fine."

# --- Website (Flutter Web served by nginx) ---

web-up: ensure-proxy-net
	$(COMPOSE) up --build -d --wait nats auth-service api-gateway web
	@$(MAKE) --no-print-directory compose-ps

web-logs:
	$(COMPOSE) logs -f --tail=100 web

web-health:
	curl -sf "$(WEB_URL)/web-healthz"
	@echo
	curl -sf "$(WEB_URL)/gateway-healthz"
	@echo
	curl -sf "$(WEB_URL)/v1/hello"
	@echo

stack-health: compose-ps health web-health

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

swagger:
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g main.go -d $(SWAG_DIRS) -o services/api-gateway/docs

swagger-check: swagger
	git diff --exit-code -- services/api-gateway/docs

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
	cd apps/mobile && flutter run -d chrome --dart-define=GOOGLE_WEB_CLIENT_ID="$(GOOGLE_WEB_CLIENT_ID)"

flutter-android:
	cd apps/mobile && flutter run -d android --dart-define=GOOGLE_WEB_CLIENT_ID="$(GOOGLE_WEB_CLIENT_ID)"

flutter-ios:
	cd apps/mobile && flutter run -d ios --dart-define=GOOGLE_WEB_CLIENT_ID="$(GOOGLE_WEB_CLIENT_ID)" --dart-define=GOOGLE_IOS_CLIENT_ID="$(GOOGLE_IOS_CLIENT_ID)"

flutter-devices:
	cd apps/mobile && flutter devices
