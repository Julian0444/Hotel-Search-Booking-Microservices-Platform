FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN go build -o app ./cmd/main.go

EXPOSE 8082

CMD ["./app"]