# ------------------------------------------------
# Stage 1: Build the Go application
# ------------------------------------------------
FROM golang:1.25.1-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install git and other build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files to take advantage of Docker layer caching
# This ensures dependencies are only re-downloaded if go.mod/go.sum changes
COPY go.mod go.sum ./

# Download dependencies. This uses the cache from the previous layer.
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the application. Use CGO_ENABLED=0 for static linking,
# which is necessary for creating a small, dependency-free binary for the final stage.
RUN go build -ldflags "-s -w" -o /authservice .


# ------------------------------------------------
# Stage 2: Create the final minimal image
# ------------------------------------------------
FROM alpine:latest

# Set the working directory for the final application
WORKDIR /root/

# Copy the static binary from the builder stage
# The name 'authservice' is the name of the output binary in the previous stage
COPY --from=builder /authservice .

# Expose the port your service runs on (assuming port 8080 or similar)
EXPOSE 8081

# The command to run the application when the container starts
# This executes the single compiled binary
CMD ["./authservice"]