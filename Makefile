# Nota: en un workspace cuya raíz no es un módulo, `go build ./...` no
# resuelve — hay que enumerar los módulos.
GO_PACKAGES = ./users-api/... ./hotels-api/... ./search-api/... ./platform-contracts/...
GO_MODULES  = users-api hotels-api search-api platform-contracts

.PHONY: build test test-integration lint fmt up up-frontend down seed frontend-check e2e

build:
	go build $(GO_PACKAGES)

test:
	go test -race $(GO_PACKAGES)

test-integration:
	cd hotels-api && go test -tags=integration ./...

lint:
	@for m in $(GO_MODULES); do \
		echo "== golangci-lint $$m"; \
		(cd $$m && golangci-lint run ./...) || exit 1; \
	done

fmt:
	gofmt -w ./users-api ./hotels-api ./search-api ./platform-contracts

up:
	docker compose up -d --build

up-frontend:
	docker compose --profile frontend up -d --build

down:
	docker compose down -v

# Frontend (plan 13): lint + unit/component con coverage + build de producción
frontend-check:
	cd frontend && npm run check

# E2E Playwright contra el stack real (plan 13 fase 10). Requiere los certs
# TLS locales (nginx/certs/README.md) y npx playwright install chromium.
# Las credenciales admin las leen los specs del .env de la raíz.
e2e: up-frontend
	cd frontend && npm run test:e2e

seed:
	docker compose run --rm migrate
	@echo "Usuarios demo -> cliente: demo/DemoCliente123 (migración 0002) · admin: ADMIN_USERNAME/ADMIN_PASSWORD del .env (seed al arranque)"
	@echo "Hoteles demo -> mongo-init.js corre solo con volumen nuevo: make down && make up para re-sembrar"
