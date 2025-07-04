FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o keyvalue-server ./cmd/server

# Use a smaller image for the final build
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create a non-root user to run the application
RUN adduser -D -g '' keyvalue

# Create necessary directories
RUN mkdir -p /data && chown -R keyvalue:keyvalue /data

# Copy the binaries from the builder stage
COPY --from=builder /app/keyvalue-server /usr/local/bin/

# Set working directory
WORKDIR /cmd

# Use the non-root user
USER keyvalue

# Expose the HTTP and Raft ports
EXPOSE 8080 7000 6379

CMD [". /keyvalue"]