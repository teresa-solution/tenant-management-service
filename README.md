# Tenant Management Service

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.20+-00ADD8.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)
![Coverage](https://img.shields.io/badge/coverage-85%25-green.svg)

A robust, secure, and scalable multi-tenant management service built with Go, designed for SaaS applications.

## 🌟 Features

- **Secure Tenant Isolation**: Each tenant gets their own database schema
- **Data Encryption**: Contact emails are encrypted at rest
- **Connection Pooling**: Efficient database connection management with connection-pool-manager service
- **Redis Caching**: High-performance caching layer for tenant data
- **Metrics & Monitoring**: Built-in Prometheus metrics
- **TLS Security**: Secure communication between services with TLS
- **Async Provisioning**: Background tenant provisioning workflow
- **Soft Delete**: Non-destructive tenant removal
- **Health Checks**: Built-in service health monitoring

## 🏗️ Architecture

```
┌─────────────────┐      ┌───────────────────┐      ┌──────────────┐
│    API Client   │─────▶│ Tenant Management │─────▶│ Redis Cache  │
│                 │◀─────│     Service       │◀─────│              │
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
./tenant-management-service --port=50051 --db-host=localhost --db-port=5432 --db-user=admin --db-pass=securepassword --db-name=tenant_registry
```

## 📦 API Reference

The service exposes a gRPC API with the following methods:

### CreateTenant

Creates a new tenant with proper validation and begins the provisioning process.

```protobuf
rpc CreateTenant(CreateTenantRequest) returns (CreateTenantResponse);
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
| `--db-host` | Database host | localhost |
| `--db-port` | Database port | 5432 |
| `--db-user` | Database username | admin |
| `--db-pass` | Database password | securepassword |
| `--db-name` | Database name | tenant_registry |

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
