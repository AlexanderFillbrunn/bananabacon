# Stage 1: Build the application
FROM golang:1.23-alpine AS builder

# Set the working directory
WORKDIR /app

# Install dependencies required for the build
RUN apk add --no-cache git

# Copy go.mod and go.sum files for dependency installation
COPY go.mod go.sum ./

# Download and cache Go modules
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o server ./cmd

# Stage 2: Create a minimal image for running the server
FROM alpine:latest

ENV METRICS_PORT 8080

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Expose the port the server listens on (e.g., 8080)
EXPOSE 8080

# Command to run the application
CMD ["./server"]
