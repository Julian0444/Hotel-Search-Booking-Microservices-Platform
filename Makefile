# Nota: en un workspace cuya raíz no es un módulo, `go build ./...` no
# resuelve — hay que enumerar los módulos.
GO_PACKAGES = ./users-api/... ./hotels-api/... ./search-api/... ./platform-contracts/...
GO_MODULES  = users-api hotels-api search-api platform-contracts

.PHONY: build test test-integration lint fmt up down seed

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

down:
	docker compose down -v

seed:
	docker compose run --rm migrate
	@echo "Usuarios demo -> cliente: demo/DemoCliente123 (migración 0002) · admin: ADMIN_USERNAME/ADMIN_PASSWORD del .env (seed al arranque)"
	@echo "Hoteles demo -> mongo-init.js corre solo con volumen nuevo: make down && make up para re-sembrar"
