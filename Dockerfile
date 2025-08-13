FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

#############################
# STEP 2 build a small image
##############################
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates wireguard-tools iptables

# Create non-root user
# RUN adduser -D -s /bin/sh vpnuser

WORKDIR /root

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose API ports
EXPOSE 8080
EXPOSE 3000

# Run as non-root user for security
# USER vpnuser

# Command to run
CMD ["./main"]
