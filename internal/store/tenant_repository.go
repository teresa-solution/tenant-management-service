package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/teresa-solution/tenant-management-service/internal/model"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(dsn string) (*TenantRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &TenantRepository{db: db}, nil
}

func (r *TenantRepository) Close() error {
	return r.db.Close()
}

func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	query := `INSERT INTO tenants (id, name, subdomain, status, provisioned, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	tenant.ID = uuid.New()
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = tenant.CreatedAt
	_, err := r.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Subdomain, tenant.Status, tenant.Provisioned, tenant.CreatedAt, tenant.UpdatedAt)
	return err
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	query := `SELECT id, name, subdomain, status, provisioned, created_at, updated_at, deleted_at
              FROM tenants WHERE id = $1`
	tenant := &model.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.Status, &tenant.Provisioned, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tenant, err
}

func (r *TenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	query := `UPDATE tenants SET name = $2, subdomain = $3, status = $4, provisioned = $5, updated_at = $6
              WHERE id = $1`
	tenant.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Subdomain, tenant.Status, tenant.Provisioned, tenant.UpdatedAt)
	return err
}

func (r *TenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	query := `SELECT id, name, subdomain, status, provisioned, created_at, updated_at, deleted_at
              FROM tenants WHERE subdomain = $1`
	tenant := &model.Tenant{}
	err := r.db.QueryRowContext(ctx, query, subdomain).Scan(&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.Status, &tenant.Provisioned, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tenant, err
}

func (r *TenantRepository) CreateProvisioningLog(ctx context.Context, tenantID uuid.UUID, step, status string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}
	query := `INSERT INTO tenant_provisioning_logs (tenant_id, step, status, details, created_at)
              VALUES ($1, $2, $3, $4, $5)`
	_, err = r.db.ExecContext(ctx, query, tenantID, step, status, detailsJSON, time.Now())
	return err
}

func (r *TenantRepository) GetTenantSchema(ctx context.Context, tenantID uuid.UUID) (string, error) {
	query := `SELECT schema_name FROM tenant_schemas WHERE tenant_id = $1`
	var schemaName string
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&schemaName)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("schema not found for tenant %s", tenantID)
	}
	return schemaName, err
}

func (r *TenantRepository) CreateTenantSchema(ctx context.Context, tenantID uuid.UUID, subdomain string) error {
	query := `SELECT create_tenant_schema($1, $2)`
	_, err := r.db.ExecContext(ctx, query, tenantID, subdomain)
	return err
}
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return err
	}
	// Verify the update affected a row
	var count int
	countQuery := `SELECT COUNT(*) FROM tenants WHERE id = $1 AND deleted_at IS NOT NULL`
	err = r.db.QueryRowContext(ctx, countQuery, id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("tenant not found or already deleted")
	}
	return nil
}
