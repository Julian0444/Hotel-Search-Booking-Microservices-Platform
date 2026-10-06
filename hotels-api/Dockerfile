# Multi-stage / no-root (I2). Build context = raíz del repo (ver
# docker-compose.yml): el replace de go.mod necesita ./platform-contracts
# dentro del contexto.
#
# Bases pineadas por digest del MANIFEST-LIST (CN4), no de una arquitectura
# suelta — un digest por-arch rompe el build multi-arch del CI.
# Refrescar con: docker buildx imagetools inspect <imagen:tag>
#
# --platform=$BUILDPLATFORM: el builder corre nativo en el host y CROSS-compila
# a $TARGETOS/$TARGETARCH — sin emular el toolchain de Go bajo QEMU.
FROM --platform=$BUILDPLATFORM golang:1.26.8-alpine3.24@sha256:8ac98ca534ac3f51e1f420a1dd2c15e74c75cfa0f23f3ad27eb5d7236c349a0c AS builder

WORKDIR /src

# Módulo de contratos compartido (resuelto vía replace ../platform-contracts)
COPY platform-contracts/ ./platform-contracts/

# Dependencias primero para cache de Docker
COPY hotels-api/go.mod hotels-api/go.sum ./hotels-api/
WORKDIR /src/hotels-api
RUN go mod download

# Copiar código y compilar (entrypoint en cmd/main.go)
COPY hotels-api/ .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/hotels-api ./cmd/main.go

# Production stage
FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiar binario del stage anterior
COPY --from=builder /out/hotels-api .

# Usuario no-root con UID FIJO y USER numérico: el runAsNonRoot de k8s no
# puede verificar un user por nombre (CreateContainerConfigError)
RUN adduser -D -g '' -u 10001 appuser
USER 10001:10001

EXPOSE 8081

CMD ["./hotels-api"]
