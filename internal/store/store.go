package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	pb "github.com/teresa-solution/connection-pool-manager/proto"
	"github.com/teresa-solution/tenant-management-service/internal/crypto"
	"github.com/teresa-solution/tenant-management-service/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Close() error
}

type TenantRepository struct {
	connMgrClient pb.ConnectionPoolServiceClient
	redis         RedisClient // Updated to use the interface
	connections   map[string]*pgxpool.Pool
	mutex         sync.Mutex
}

func NewTenantRepository(dsn string) (*TenantRepository, error) {
	creds, err := credentials.NewClientTLSFromFile("certs/cert.pem", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS credentials: %v", err)
	}
	conn, err := grpc.Dial("connection-pool-manager:50052", grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to dial connection-pool-manager: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return &TenantRepository{
		connMgrClient: pb.NewConnectionPoolServiceClient(conn),
		redis:         rdb, // This still works because redis.Client implements RedisClient
		connections:   make(map[string]*pgxpool.Pool),
	}, nil
}

func (r *TenantRepository) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for connID, pool := range r.connections {
		pool.Close()
		delete(r.connections, connID)
	}
	return r.redis.Close()
}

// getPool requests a connection pool from the connection-pool-manager
func (r *TenantRepository) getPool(ctx context.Context) (string, *pgxpool.Pool, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// For simplicity, use a static tenant ID and DSN
	tenantID := "tenant-management-service"
	req := &pb.ConnectionRequest{
		TenantId: tenantID,
		Dsn:      "host=localhost port=5432 user=admin password=securepassword dbname=tenant_registry",
	}
	resp, err := r.connMgrClient.GetConnection(ctx, req)
	if err != nil || resp.Error != "" {
		return "", nil, fmt.Errorf("failed to get connection: %v, error: %s", err, resp.Error)
	}

	// Check if we already have a pool for this ConnectionId
	connID := resp.ConnectionId
	if pool, exists := r.connections[connID]; exists {
		return connID, pool, nil
	}

	// Create a new pool (in reality, the connection-pool-manager should provide the pool or DSN)
	config, err := pgxpool.ParseConfig(req.Dsn)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse DSN: %v", err)
	}
	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create connection pool: %v", err)
	}

	r.connections[connID] = pool
	return connID, pool, nil
}

// releasePool releases the connection back to the connection-pool-manager
func (r *TenantRepository) releasePool(ctx context.Context, connID string) error {
	if connID == "" {
		return nil
	}
	_, err := r.connMgrClient.ReleaseConnection(ctx, &pb.ConnectionRelease{ConnectionId: connID})
	if err != nil {
		return fmt.Errorf("failed to release connection: %v", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if pool, exists := r.connections[connID]; exists {
		pool.Close()
		delete(r.connections, connID)
	}
	return nil
}

func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return err
	}
	defer r.releasePool(ctx, connID)

	tenant.ID = uuid.New()
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = tenant.CreatedAt

	// Encrypt email if provided
	if tenant.ContactEmail != "" {
		encryptedEmail, iv, err := crypto.Encrypt(tenant.ContactEmail)
		if err != nil {
			return err
		}
		tenant.EncryptedEmail = encryptedEmail
		tenant.EmailIV = iv
	}

	query := `INSERT INTO tenants (id, name, subdomain, encrypted_email, email_iv, status, provisioned, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = pool.Exec(ctx, query, tenant.ID, tenant.Name, tenant.Subdomain, tenant.EncryptedEmail, tenant.EmailIV, tenant.Status, tenant.Provisioned, tenant.CreatedAt, tenant.UpdatedAt)
	if err != nil {
		return err
	}

	// Invalidate cache for this tenant
	r.redis.Del(ctx, fmt.Sprintf("tenant:%s", tenant.ID.String()))
	return nil
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
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return nil, err
	}
	defer r.releasePool(ctx, connID)

	query := `SELECT id, name, subdomain, encrypted_email, email_iv, status, provisioned, created_at, updated_at, deleted_at
              FROM tenants WHERE id = $1`
	tenant := &model.Tenant{}
	err = pool.QueryRow(ctx, query, id).Scan(&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.EncryptedEmail, &tenant.EmailIV, &tenant.Status, &tenant.Provisioned, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt)
	if err != nil && err.Error() == "no rows in result set" {
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
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return err
	}
	defer r.releasePool(ctx, connID)

	// Encrypt email if provided
	if tenant.ContactEmail != "" {
		encryptedEmail, iv, err := crypto.Encrypt(tenant.ContactEmail)
		if err != nil {
			return err
		}
		tenant.EncryptedEmail = encryptedEmail
		tenant.EmailIV = iv
	}

	query := `UPDATE tenants SET name = $2, subdomain = $3, encrypted_email = $4, email_iv = $5, status = $6, provisioned = $7, updated_at = $8
              WHERE id = $1`
	tenant.UpdatedAt = time.Now()
	_, err = pool.Exec(ctx, query, tenant.ID, tenant.Name, tenant.Subdomain, tenant.EncryptedEmail, tenant.EmailIV, tenant.Status, tenant.Provisioned, tenant.UpdatedAt)
	if err != nil {
		return err
	}

	// Invalidate cache
	r.redis.Del(ctx, fmt.Sprintf("tenant:%s", tenant.ID.String()))
	return nil
}

func (r *TenantRepository) GetBySubdomain(ctx context.Context, subdomain string) (*model.Tenant, error) {
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return nil, err
	}
	defer r.releasePool(ctx, connID)

	query := `SELECT id, name, subdomain, encrypted_email, email_iv, status, provisioned, created_at, updated_at, deleted_at
              FROM tenants WHERE subdomain = $1`
	tenant := &model.Tenant{}
	err = pool.QueryRow(ctx, query, subdomain).Scan(&tenant.ID, &tenant.Name, &tenant.Subdomain, &tenant.EncryptedEmail, &tenant.EmailIV, &tenant.Status, &tenant.Provisioned, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt)
	if err != nil && err.Error() == "no rows in result set" {
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
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return err
	}
	defer r.releasePool(ctx, connID)

	query := `UPDATE tenants SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err = pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	var count int
	countQuery := `SELECT COUNT(*) FROM tenants WHERE id = $1 AND deleted_at IS NOT NULL`
	err = pool.QueryRow(ctx, countQuery, id).Scan(&count)
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
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return err
	}
	defer r.releasePool(ctx, connID)

	query := `SELECT create_tenant_schema($1, $2)`
	_, err = pool.Exec(ctx, query, tenantID, subdomain)
	if err != nil {
		return err
	}

	// Invalidate cache if tenant exists
	r.redis.Del(ctx, fmt.Sprintf("tenant:%s", tenantID.String()))
	return nil
}

func (r *TenantRepository) GetTenantSchema(ctx context.Context, tenantID uuid.UUID) (string, error) {
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return "", err
	}
	defer r.releasePool(ctx, connID)

	query := `SELECT schema_name FROM tenant_schemas WHERE tenant_id = $1`
	var schemaName string
	err = pool.QueryRow(ctx, query, tenantID).Scan(&schemaName)
	if err != nil && err.Error() == "no rows in result set" {
		return "", fmt.Errorf("schema not found for tenant %s", tenantID)
	}
	return schemaName, err
}

func (r *TenantRepository) CreateProvisioningLog(ctx context.Context, tenantID uuid.UUID, step, status string, details interface{}) error {
	connID, pool, err := r.getPool(ctx)
	if err != nil {
		return err
	}
	defer r.releasePool(ctx, connID)

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}
	query := `INSERT INTO tenant_provisioning_logs (tenant_id, step, status, details, created_at)
              VALUES ($1, $2, $3, $4, $5)`
	_, err = pool.Exec(ctx, query, tenantID, step, status, detailsJSON, time.Now())
	return err
}
