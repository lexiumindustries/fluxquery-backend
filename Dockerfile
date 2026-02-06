# Build stage
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /export-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /export-service .
# Copy .env.example as a template if .env is missing (though prod should use env vars)
COPY .env.example .env

# Expose the configured port
EXPOSE 8080

# Run the binary
CMD ["./export-service"]
