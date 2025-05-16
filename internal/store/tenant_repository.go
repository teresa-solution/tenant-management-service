package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/teresa-solution/tenant-management-service/internal/crypto"
	"github.com/teresa-solution/tenant-management-service/internal/model"
)

type TenantRepository struct {
	db    *sql.DB
	redis *redis.Client
}

func NewTenantRepository(dsn string) (*TenantRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password by default
		DB:       0,  // Use default DB
	})

	return &TenantRepository{db: db, redis: rdb}, nil
}

func (r *TenantRepository) Close() error {
	if err := r.db.Close(); err != nil {
		return err
	}
	return r.redis.Close()
}

func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	query := `INSERT INTO tenants (id, name, subdomain, encrypted_email, email_iv, status, provisioned, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	tenant.ID = uuid.New()
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = tenant.CreatedAt
	_, err := r.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Subdomain, tenant.EncryptedEmail, tenant.EmailIV, tenant.Status, tenant.Provisioned, tenant.CreatedAt, tenant.UpdatedAt)
	if err == nil {
		// Invalidate cache for this tenant (if it exists)
		r.redis.Del(ctx, fmt.Sprintf("tenant:%s", tenant.ID.String()))
	}
	return err
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	// Check cache first
	key := fmt.Sprintf("tenant:%s", id.String())
	cached, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		tenant := &model.Tenant{}
		if err := json.Unmarshal([]byte(cached), tenant); err == nil {
			return tenant, nil
		}
	}

	// Cache miss, query database
	query := `SELECT id, name, subdomain, encrypted_email, email_iv, status, provisioned, created_at, updated_at, deleted_at
              FROM tenants WHERE id = $1`
	tenant := &model.Tenant{}
	err = r.db.QueryRowContext(ctx, query, id).Scan(&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.EncryptedEmail, &tenant.EmailIV, &tenant.Status, &tenant.Provisioned, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt email if encrypted
	if len(tenant.EncryptedEmail) > 0 && len(tenant.EmailIV) > 0 {
		contactEmail, err := crypto.Decrypt(tenant.EncryptedEmail, tenant.EmailIV)
		if err != nil {
			return nil, err
		}
		tenant.ContactEmail = contactEmail
	}

	// Cache the result
	data, err := json.Marshal(tenant)
	if err == nil {
		r.redis.SetEx(ctx, key, data, 1*time.Hour) // Cache for 1 hour
	}

	return tenant, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	query := `UPDATE tenants SET name = $2, subdomain = $3, encrypted_email = $4, email_iv = $5, status = $6, provisioned = $7, updated_at = $8
              WHERE id = $1`
	tenant.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query, tenant.ID, tenant.Name, tenant.Subdomain, tenant.EncryptedEmail, tenant.EmailIV, tenant.Status, tenant.Provisioned, tenant.UpdatedAt)
	if err == nil {
		// Invalidate cache
		r.redis.Del(ctx, fmt.Sprintf("tenant:%s", tenant.ID.String()))
	}
	return err
}

func (r *TenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	query := `SELECT id, name, subdomain, encrypted_email, email_iv, status, provisioned, created_at, updated_at, deleted_at
              FROM tenants WHERE subdomain = $1`
	tenant := &model.Tenant{}
	err := r.db.QueryRowContext(ctx, query, subdomain).Scan(&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.EncryptedEmail, &tenant.EmailIV, &tenant.Status, &tenant.Provisioned, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt email if encrypted
	if len(tenant.EncryptedEmail) > 0 && len(tenant.EmailIV) > 0 {
		contactEmail, err := crypto.Decrypt(tenant.EncryptedEmail, tenant.EmailIV)
		if err != nil {
			return nil, err
		}
		tenant.ContactEmail = contactEmail
	}

	return tenant, nil
}

func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return err
	}
	var count int
	countQuery := `SELECT COUNT(*) FROM tenants WHERE id = $1 AND deleted_at IS NOT NULL`
	err = r.db.QueryRowContext(ctx, countQuery, id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("tenant not found or already deleted")
	}
	// Invalidate cache
	r.redis.Del(ctx, fmt.Sprintf("tenant:%s", id.String()))
	return nil
}

func (r *TenantRepository) CreateTenantSchema(ctx context.Context, tenantID uuid.UUID, subdomain string) error {
	query := `SELECT create_tenant_schema($1, $2)`
	_, err := r.db.ExecContext(ctx, query, tenantID, subdomain)
	if err == nil {
		// Invalidate cache if tenant exists
		r.redis.Del(ctx, fmt.Sprintf("tenant:%s", tenantID.String()))
	}
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
