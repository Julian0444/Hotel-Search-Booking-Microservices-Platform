# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Dependencias primero para cache de Docker
COPY go.mod go.sum ./
RUN go mod download

# Copiar código y compilar
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/users-api ./cmd/main.go

# Production stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiar binario del stage anterior
COPY --from=builder /app/users-api .

# Usuario no-root
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8082

CMD ["./users-api"]
