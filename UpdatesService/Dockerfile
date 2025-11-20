# ------------------------------------------------
# Stage 1: Build the Go application with CGO
# ------------------------------------------------
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install build dependencies: git for modules, build-base/pkgconfig/librdkafka-dev for CGO/Kafka
RUN apk update && apk add --no-cache \
    git \
    build-base \
    pkgconfig \
    librdkafka-dev \
    ca-certificates

# Copy go.mod and go.sum to leverage Docker layer caching
COPY go.mod go.sum ./

# Download dependencies (including confluent-kafka-go, mongo-driver, go-redis)
RUN go mod download

# Copy the entire source code
COPY . .

# Build the application. 
# - CGO_ENABLED=1: Mandatory for confluent-kafka-go
# - -tags musl: Tells the compiler we are building for Alpine (musl libc)
# - -o /updates_service .: Build the package in the current directory (.)
RUN CGO_ENABLED=1 go build -tags musl -ldflags "-s -w" -o /updates_service .


# ------------------------------------------------
# Stage 2: Create the final minimal runtime image
# ------------------------------------------------
FROM alpine:latest

# Install runtime dependencies for the Kafka C library (librdkafka)
# libstdc++ is often required by C++ libraries like librdkafka
RUN apk update && apk add --no-cache \
    libstdc++ \
    librdkafka \
    ca-certificates

# Set the working directory for the final application
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /updates_service .

# The Updates Service is the entry point for clients (WebSocket/HTTP), 
# so it MUST expose the port (e.g., 8083 based on your previous snippets).

EXPOSE 8083 

# The command to run the application
CMD ["./updates_service"]