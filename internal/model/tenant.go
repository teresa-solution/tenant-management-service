package model

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents the tenants table
type Tenant struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Subdomain      string     `json:"subdomain"`
	ContactEmail   string     // Plaintext (transient, not stored in DB)
	EncryptedEmail []byte     // Stored in DB
	EmailIV        []byte     // Stored in DB
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	Provisioned    bool       `db:"provisioned"` // New field
}

// TenantContact represents the tenant_contacts table
type TenantContact struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TenantDatabaseConfig represents the tenant_database_configs table
type TenantDatabaseConfig struct {
	ID                        uuid.UUID `json:"id"`
	TenantID                  uuid.UUID `json:"tenant_id"`
	Host                      string    `json:"host"`
	Port                      int       `json:"port"`
	DatabaseName              string    `json:"database_name"`
	SchemaName                string    `json:"schema_name"`
	Username                  string    `json:"username"`
	PasswordSecretID          string    `json:"password_secret_id"`
	MaxConnections            int       `json:"max_connections"`
	IdleConnections           int       `json:"idle_connections"`
	ConnectionLifetimeMinutes int       `json:"connection_lifetime_minutes"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}
