# 🏢 Teresa Tenant Management Service

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.20+-00ADD8.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage](https://img.shields.io/badge/coverage-85%25-green.svg)

A robust, secure, and scalable multi-tenant management service built with Go, designed for SaaS applications within the Teresa Solution ecosystem.

## 🌟 Features

- **Secure Tenant Isolation**: Each tenant gets their own database schema
- **Data Encryption**: Contact emails are encrypted at rest
- **Connection Pooling**: Efficient database connection management via the Connection Pool Manager
- **Redis Caching**: High-performance caching layer for tenant data
- **Metrics & Monitoring**: Built-in Prometheus metrics
- **TLS Security**: Secure communication between services with TLS
- **Async Provisioning**: Background tenant provisioning workflow
- **Soft Delete**: Non-destructive tenant removal
- **Health Checks**: Built-in service health monitoring

## 🧩 Teresa Ecosystem Integration

The Tenant Management Service is a core part of the Teresa Solution platform:

* Accessed through the **[Teresa API Gateway](https://github.com/teresa-solution/api-gateway)** for client requests
* Utilizes the **[Connection Pool Manager](https://github.com/teresa-solution/connection-pool-manager)** for efficient database connection management

## 🏗️ Architecture

```
┌─────────────────┐      ┌───────────────────┐      ┌──────────────┐
│                 │      │ Tenant Management │      │              │
│  API Gateway    │─────▶│     Service       │─────▶│ Redis Cache  │
│                 │◀─────│                   │◀─────│              │
└─────────────────┘      └───────────────────┘      └──────────────┘
                                 │  ▲
                                 │  │
                                 ▼  │
                        ┌───────────────────┐      ┌──────────────┐
                        │ Connection Pool   │─────▶│  PostgreSQL  │
                        │    Manager        │◀─────│   Database   │
                        └───────────────────┘      └──────────────┘
```

## 🚀 Getting Started

### Prerequisites

- Go 1.20 or higher
- PostgreSQL 14+
- Redis 6+
- TLS certificates (for secure communication)
- Connection Pool Manager service running
- Teresa API Gateway (optional, for client access)

### Environment Setup

1. Generate TLS certificates:

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes
```

2. Set up the PostgreSQL database:

```bash
# Create database
createdb -U postgres tenant_registry

# Create required tables
psql -U postgres -d tenant_registry -f scripts/schema.sql
```

### Running the Service

```bash
go build -o tenant-management-service
./tenant-management-service --port=50051 --pool-mgr-addr=localhost:50052 --redis-addr=localhost:6379
```

## 📦 API Reference

The service exposes a gRPC API with the following methods:

### CreateTenant

Creates a new tenant with proper validation and begins the provisioning process.

```protobuf
rpc CreateTenant(CreateTenantRequest) returns (CreateTenantResponse);
```

Request:
```protobuf
message CreateTenantRequest {
  string name = 1;                // Company/organization name
  string subdomain = 2;           // Unique subdomain identifier
  string contact_email = 3;       // Primary contact email (encrypted at rest)
  map<string, string> metadata = 4; // Additional tenant metadata
}
```

Response:
```protobuf
message CreateTenantResponse {
  string tenant_id = 1;           // Unique tenant ID
  string status = 2;              // "provisioning" | "active" | "failed"
  string provisioning_job_id = 3; // ID for tracking the provisioning job
}
```

### GetTenant

Retrieves tenant information by ID.

```protobuf
rpc GetTenant(GetTenantRequest) returns (GetTenantResponse);
```

### UpdateTenant

Updates tenant information with validation.

```protobuf
rpc UpdateTenant(UpdateTenantRequest) returns (UpdateTenantResponse);
```

### DeleteTenant

Soft deletes a tenant.

```protobuf
rpc DeleteTenant(DeleteTenantRequest) returns (DeleteTenantResponse);
```

### Provisioning Workflow

When a new tenant is created, the service:

1. Validates input and creates tenant record
2. Initiates asynchronous provisioning with the Connection Pool Manager
3. Creates dedicated database schema for the tenant
4. Sets up initial tenant configuration
5. Updates tenant status to "active" when complete

## 🔐 Integration with Connection Pool Manager

The Tenant Management Service relies on the Connection Pool Manager for efficient database access:

```go
// Example of requesting a database connection for a tenant
conn, err := poolClient.GetConnection(ctx, &poolpb.ConnectionRequest{
    TenantId: tenantID,
    Dsn: fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
        config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName),
})

// Use the connection for tenant operations
// ...

// Release the connection when done
_, err = poolClient.ReleaseConnection(ctx, &poolpb.ConnectionRelease{
    ConnectionId: conn.ConnectionId,
})
```

## 🔒 Security Features

- **Email Encryption**: Contact emails are encrypted using AES-256
- **TLS Communication**: All service-to-service communication is encrypted
- **Secure Connection Management**: Connection pools are securely managed
- **Input Validation**: All API inputs are thoroughly validated
- **Soft Delete**: Records are never permanently removed

## 📊 Monitoring

The service exposes Prometheus metrics at `/metrics` and provides a health check endpoint at `/health`.

Key metrics tracked:
- Provisioning success/failure rates
- Provisioning duration
- Request latencies
- Connection pool utilization

## 🧪 Testing

Run the test suite with:

```bash
go test ./... -v -cover
```

## 🔧 Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | gRPC server port | 50051 |
| `--pool-mgr-addr` | Connection Pool Manager address | localhost:50052 |
| `--redis-addr` | Redis server address | localhost:6379 |
| `--metrics-port` | HTTP metrics port | 8081 |

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
