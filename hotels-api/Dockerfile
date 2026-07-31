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
FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

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
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

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
