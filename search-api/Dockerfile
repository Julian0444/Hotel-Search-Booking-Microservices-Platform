# Use the Go image with Alpine for building and running the application
# Build context = repo root (see docker-compose.yml): the go.mod replace
# directive needs ../platform-contracts inside the context.
FROM golang:1.25-alpine

# Install netcat for health checks
RUN apk add --no-cache netcat-openbsd

# Set the working directory inside the container
WORKDIR /app

# Copy the shared contracts module (resolved via replace ../platform-contracts)
COPY platform-contracts/ ./platform-contracts/

# Copy go.mod and go.sum and download dependencies
COPY search-api/go.mod search-api/go.sum ./search-api/
WORKDIR /app/search-api
RUN go mod download

# Copy the rest of the code and build the application
COPY search-api/ .
RUN go build -o app ./cmd/main.go

# Expose the port on which the app will run
EXPOSE 8082

# Command to run the binary
CMD ["./app"]
