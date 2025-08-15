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

RUN mkdir -p /etc/s6-overlay/s6-rc.d/vpn-api/serviced.d
COPY ./vpn-api.sh /etc/s6-overlay/s6-rc.d/vpn-api/run

RUN chmod +x /etc/s6-overlay/s6-rc.d/vpn-api/run
