# ------------------------------------------------
# Stage 1: Build the Go application with CGO
# ------------------------------------------------
FROM golang:1.25.1-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install dependencies necessary to compile librdkafka (the C dependency for confluent-kafka-go)
# build-base provides gcc/g++, librdkafka-dev provides header files
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

# Copy the entire source code
COPY . .

# Build the application. CGO_ENABLED=1 is CRITICAL for linking librdkafka.
# Assuming the main entry point is in ./cmd/documentservice
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o /documentservice .


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
COPY --from=builder /documentservice .

# Expose the port your service runs on (from your main.go snippet)
EXPOSE 8082

# The command to run the application
CMD ["./documentservice"]