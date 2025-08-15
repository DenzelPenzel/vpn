# VPN Service

This project is a secure, private, and censorship-resistant SaaS VPN service built with Go, WireGuard, and Docker.

## ✨ Guiding Principles

-   **Security First**: Employ modern, audited cryptography and a minimal attack surface.
-   **Privacy by Design**: Enforce a strict no-logs policy for all user activity.
-   **Performance**: Deliver high-speed connections with low latency.
-   **Developer Experience**: Maintain a simple, reproducible local development environment using Docker.

## 🛠️ Technology Stack

| Component          | Technology              | Purpose & Rationale                                                   |
| ------------------ | ----------------------- | --------------------------------------------------------------------- |
| **VPN Protocol**   | WireGuard               | Fast, modern, secure Layer 3 VPN with a minimal codebase.             |
| **Obfuscation**    | v2ray-plugin            | Wraps WireGuard UDP packets in WebSocket+TLS to evade DPI.            |
| **Backend API**    | Go (FastHTTP)           | High performance, excellent concurrency, and single binary deployment.    |
| **Web Server**     | Caddy                   | Reverse proxy with automatic local HTTPS for simplified TLS management.   |
| **Database**       | PostgreSQL              | Powerful, reliable, and open-source relational database.              |
| **Containerization** | Docker & Docker Compose | Provides a consistent and reproducible local development environment.   |

## 📦 Local Development

### Prerequisites

-   Docker
-   Docker Compose

### 1. Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

Review the `.env` file and change the default passwords and secrets.

### 2. Run the Service

Create a directory for the certificates:

```bash
mkdir -p certs
```

Generate the certificate and key:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/nginx.key \
  -out certs/nginx.crt \
  -subj "/CN=localhost"

Start all services in the background:

```bash
# Build and start all containers
docker-compose up --build -d
```

### 3. Test the Service

Check the health of the API:

```bash
# Use -k to allow Caddy's self-signed certificate
curl -k https://localhost/api/health
```

Run the integration test script:

```bash
./scripts/test-v2ray.sh
```

### Service URLs

-   **Secure HTTPS Proxy**: `https://localhost`
-   **Go API (Internal)**: `http://vpn_api:8080`
-   **WireGuard (Internal)**: `udp://wireguard:51820`

## 🚀 API Endpoints

| Method | Path                   | Description                                      | Authentication     |
| ------ | ---------------------- | ------------------------------------------------ | ------------------ |
| `POST` | `/api/users/register`  | Creates a new user account.                      | None               |
| `POST` | `/api/users/login`     | Authenticates a user and returns a JWT.          | None               |
| `GET`  | `/api/client/config`   | Generates a WireGuard `.conf` for the user.      | JWT Bearer Token   |
| `GET`  | `/api/servers/locations` | Returns a list of available VPN server locations.  | JWT Bearer Token   |
| `GET`  | `/api/health`          | Checks the health of the service.                | None               |

## 🔒 Security Model

-   **No-Logs Policy**: The service **MUST NOT** log user IP addresses, DNS queries, or traffic metadata. Logging is for application health only.
-   **Double Encryption**: All traffic is double-encrypted:
    -   **Inner Encryption**: WireGuard's ChaCha20Poly1305.
    -   **Outer Encryption**: TLS 1.3 provided by Caddy for the WebSocket tunnel.
-   **Key Management**: Client private keys are generated on the client and **NEVER** sent to the server. The server only stores the client's public key.
-   **Minimal Attack Surface**: The production instance exposes only TCP port 443. All other services are on the internal Docker network.
-   **Password Hashing**: User passwords are hashed using `bcrypt`.
