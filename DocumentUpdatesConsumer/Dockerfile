# ------------------------------------------------
# Stage 1: Build the Go application with CGO
# ------------------------------------------------
FROM golang:1.25.1-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install build dependencies for CGO (Kafka) and Redis
RUN apk update && apk add --no-cache \
    git \
    build-base \
    pkgconfig \
    librdkafka-dev \
    ca-certificates

# Copy go.mod and go.sum to leverage Docker layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code (assuming the main service is compiled from here)
COPY . .

# Build the application. CGO_ENABLED=1 is CRITICAL for linking librdkafka.
# Assuming the main entry point for the consumer is in ./cmd/updatesconsumer
RUN CGO_ENABLED=1 go build -tags musl -ldflags "-s -w" -o /updatesconsumer . 


# ------------------------------------------------
# Stage 2: Create the final minimal runtime image
# ------------------------------------------------
FROM alpine:latest

# Install runtime dependencies for the Kafka client library and Go binary
# This includes the shared C library and networking tools
RUN apk update && apk add --no-cache \
    libstdc++ \
    librdkafka \
    ca-certificates

# Set the working directory for the final application
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /updatesconsumer .

# The Updates Consumer Service often doesn't need to expose a port, 
# as it runs continuously in the background, consuming from Kafka.
# If it had a health check endpoint, you would EXPOSE that port.


# The command to run the application
CMD ["./updatesconsumer"]