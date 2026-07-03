# ==========================================
# Stage 1: Build Stage
# ==========================================
FROM golang:alpine AS builder

# Set working directory inside the build container
WORKDIR /app

# Install git and certificates required for dependency fetching and HTTPS calls
RUN apk add --no-cache git ca-certificates tzdata

# Copy module files first to leverage Docker layer caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go binary with optimizations:
# - CGO_ENABLED=0: Creates a statically linked binary without C dependencies
# - -ldflags="-s -w": Strips debugging information and symbol tables to minimize binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/viking-api main.go

# ==========================================
# Stage 2: Minimal Production Stage
# ==========================================
FROM alpine:latest

# Install timezone data and CA certificates for secure communications
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy the compiled static binary from the builder stage
COPY --from=builder /app/viking-api .

# Ensure upload directory exists with proper execution permissions
RUN mkdir -p /root/uploads && chmod 777 /root/uploads

# Expose the isolated internal port (for documentation purposes only)
EXPOSE 65298

# Set entrypoint to run the binary
ENTRYPOINT ["./viking-api"]
