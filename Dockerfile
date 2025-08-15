# Use a multi-stage build to compile the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/vpn_api ./cmd/server

# --- Final Stage ---
FROM linuxserver/wireguard:latest

COPY --from=builder /app/vpn_api /usr/bin/vpn_api

# Place the service script in a permanent location that is not affected by the /config volume mount.
RUN mkdir -p /etc/services.d/vpn-api
COPY ./vpn-api.sh /etc/services.d/vpn-api/run
RUN chmod +x /etc/services.d/vpn-api/run

# Create a symbolic link from the location the LSIO entrypoint expects to our permanent script location.
# This ensures the service is found even after the volume is mounted over /config.
RUN mkdir -p /config/custom-services.d
RUN ln -s /etc/services.d/vpn-api /config/custom-services.d/vpn-api
