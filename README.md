# Authvora

AI-native authentication infrastructure platform for modern applications and AI agents.

## Architecture

```
┌───────────────────────────┐     ┌──────────────────┐
│  Web/Mobile Apps          │     │    AI Agents     │
│  (Java/Node/Go/Python SDK)│     │  (MCP Server)    │
└────────┬──────────────────┘     └────────┬─────────┘
         │                                 │
         ▼                                 ▼
┌──────────────────────────────────────────────────────┐
│              API Gateway (:8080)                     │
│  Rate Limiting · Tenant Resolution · JWT             │
└───────┬─────────────────┬──────────────────┬─────────┘
        ▼                 ▼                  ▼
┌──────────────┐ ┌─────────────────┐ ┌────────────────┐
│ Auth Service │ │  Policy Engine  │ │   Risk Engine  │
│   (:8081)    │ │    (:8083)      │ │    (:8082)     │
│              │ │                 │ │                │
│     Go       │ │       Go        │ │     Python     │
└──────┬───────┘ └─────┬───────────┘ └─────┬──────────┘
       │               │                   │
       ▼               ▼                   ▼
┌─────────────────────────────────────────────┐
│     PostgreSQL          Redis               │
└─────────────────────────────────────────────┘
```

## Services


| Service       | Language   | Port | Description                                    |
| ------------- | ---------- | ---- | ---------------------------------------------- |
| api-gateway   | Go         | 8080 | Request routing, rate limiting, JWT validation |
| auth-service  | Go         | 8081 | Registration, login, OAuth, JWT, sessions      |
| risk-engine   | Python     | 8082 | AI risk scoring, anomaly detection             |
| policy-engine | Go         | 8083 | Tenant policies, MFA rules                     |
| mcp-server    | TypeScript | 8084 | AI agent interface (MCP protocol)              |


## Local Setup

### Prerequisites


| Tool           | Minimum Version | Check                    |
| -------------- | --------------- | ------------------------ |
| Docker         | 20+             | `docker --version`       |
| Docker Compose | 2.0+            | `docker-compose version` |
| Go             | 1.21+           | `go version`             |
| Python         | 3.11+           | `python3 --version`      |
| Node.js        | 20+             | `node --version`         |
| pip            | any             | `pip3 --version`         |


### Step 1: Clone and configure

```bash
cd authvora
cp .env.example .env
```

The `.env` file contains default credentials that work out of the box for local development. No changes needed.

### Step 2: Start PostgreSQL and Redis

```bash
make up-infra
```

This pulls and starts two containers:

- **authvora-postgres** on port `5432` (auto-runs the SQL migration on first boot)
- **authvora-redis** on port `6379`

Verify they're healthy:

```bash
docker ps
```

You should see both containers with status `Up ... (healthy)`.

### Step 3: Verify the database

The migration auto-creates 14 tables and seeds a default tenant. Confirm:

```bash
make db-shell
# then inside psql:
\dt
SELECT id, name, slug FROM tenants;
# exit with \q
```

You should see the `Default Tenant` with ID `a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11`. This tenant ID is used in all API requests below.

### Step 4: Start the Auth Service

```bash
make run-auth
```

This starts the core authentication service on **[http://localhost:8081](http://localhost:8081)**. Keep this terminal open.

### Step 5: Test the API

Open a new terminal and run these commands.

**Register a user:**

```bash
curl -s -X POST http://localhost:8081/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" \
  -d '{
    "email": "alice@example.com",
    "password": "securepassword123",
    "name": "Alice Smith"
  }' | python3 -m json.tool
```

Expected response: a JSON object with `user`, `access_token`, `refresh_token`, and `expires_in`.

**Login:**

```bash
curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" \
  -d '{
    "email": "alice@example.com",
    "password": "securepassword123"
  }' | python3 -m json.tool
```

**Get JWKS (public keys for token verification):**

```bash
curl -s http://localhost:8081/auth/jwks \
  -H "X-Tenant-ID: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" | python3 -m json.tool
```

**Refresh a token** (use the `refresh_token` from the login response):

```bash
curl -s -X POST http://localhost:8081/auth/refresh \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" \
  -d '{"refresh_token": "<paste_refresh_token_here>"}' | python3 -m json.tool
```

**Wrong password (should return 401):**

```bash
curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" \
  -d '{"email": "alice@example.com", "password": "wrongpassword"}' | python3 -m json.tool
```

### Step 6: Start the API Gateway (optional)

In a new terminal:

```bash
make run-gateway
```

The gateway runs on **[http://localhost:8080](http://localhost:8080)** and proxies to the backend services. With the gateway running, you can use the `/api/v1/` prefixed routes:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11" \
  -d '{"email": "bob@example.com", "password": "securepass123", "name": "Bob Jones"}' | python3 -m json.tool
```

### Step 7: Start other services (optional)

Each service runs in its own terminal:

```bash
# Risk Engine (Python) on :8082
make run-risk

# Policy Engine (Go) on :8083
make run-policy

# MCP Server (TypeScript) on :8084
make run-mcp
```

For the Risk Engine, install Python dependencies first:

```bash
cd services/risk-engine
pip3 install -r requirements.txt
cd ../..
make run-risk
```

For the MCP Server, install Node dependencies first:

```bash
cd services/mcp-server
npm install
cd ../..
make run-mcp
```

### Stopping everything

```bash
# Stop all Docker containers
make down

# Stop and remove all data (clean slate)
make down-clean
```

### Resetting the database

```bash
make db-reset
```

This drops and recreates the PostgreSQL volume. The migration re-runs automatically on next startup.

### All Makefile commands

```bash
make help
```

### Using the Node SDK

```typescript
import { Authvora } from "@authvora/node";

const authvora = new Authvora({
  baseUrl: "http://localhost:8080",
  tenantId: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
});

// Register
const result = await authvora.auth.register({
  email: "user@example.com",
  password: "securepassword123",
  name: "John Doe",
});

// Login
const session = await authvora.auth.login({
  email: "user@example.com",
  password: "securepassword123",
});

// Express middleware
app.use(authvora.middleware());
```

## MCP Tools (AI Agent Integration)

The MCP server exposes these tools for AI agents:

- `verify_token` - Verify a JWT access token
- `create_user` - Create a new user account
- `get_user` - Get user details
- `get_risk_score` - Evaluate login risk
- `simulate_attack` - Run attack simulation
- `get_login_metrics` - Get login statistics
- `list_suspicious_ips` - List flagged IPs

## Project Structure

```
authvora/
├── docs/                    # Architecture documentation
│   ├── architecture.md
│   ├── database-schema.md
│   └── api-design.md
├── services/
│   ├── auth-service/        # Go - Core authentication
│   ├── api-gateway/         # Go - API gateway
│   ├── risk-engine/         # Python - Risk scoring
│   ├── policy-engine/       # Go - Policy management
│   └── mcp-server/          # TypeScript - MCP server
├── sdk/
│   └── node-sdk/            # TypeScript - Node.js SDK
├── infra/
│   ├── docker/              # Docker Compose
│   └── migrations/          # SQL migrations
├── .env.example
├── Makefile
└── README.md
```

## Documentation

- [Architecture](docs/architecture.md) - System design and service boundaries
- [Database Schema](docs/database-schema.md) - PostgreSQL schema and Redis patterns
- [API Design](docs/api-design.md) - REST API contracts and error codes

