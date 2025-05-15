package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	_ "github.com/lib/pq"
	"github.com/teresa-solution/tenant-management-service/internal/model"
)

// TenantRepository handles database operations for tenants
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository creates a new TenantRepository
func NewTenantRepository(dsn string) (*TenantRepository, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDB(*config)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &TenantRepository{db: db}, nil
}

// Close closes the database connection
func (r *TenantRepository) Close() error {
	return r.db.Close()
}

// Create inserts a new tenant into the database
func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, subdomain, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	tenant.ID = uuid.New()
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = tenant.CreatedAt
	tenant.Status = "provisioning"

	err := r.db.QueryRowContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Subdomain, tenant.Status,
		tenant.CreatedAt, tenant.UpdatedAt,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
	return err
}

// GetByID retrieves a tenant by ID
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	query := `
		SELECT id, name, subdomain, status, created_at, updated_at, deleted_at
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`
	tenant := &model.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.Status,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// GetBySubdomain retrieves a tenant by subdomain
func (r *TenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	query := `
		SELECT id, name, subdomain, status, created_at, updated_at, deleted_at
		FROM tenants
		WHERE subdomain = $1 AND deleted_at IS NULL
	`
	tenant := &model.Tenant{}
	err := r.db.QueryRowContext(ctx, query, subdomain).Scan(
		&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.Status,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// Update updates a tenant in the database
func (r *TenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, subdomain = $3, status = $4, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		tenant.ID, tenant.Name, tenant.Subdomain, tenant.Status,
	).Scan(&tenant.UpdatedAt)
	return err
}

// Delete performs a soft delete on a tenant
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE tenants
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *TenantRepository) CreateProvisioningLog(ctx context.Context, tenantID uuid.UUID, step, status string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	query := `INSERT INTO tenant_provisioning_logs (tenant_id, step, status, details, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = r.db.ExecContext(ctx, query, tenantID, step, status, detailsJSON, time.Now())
	return err
}
