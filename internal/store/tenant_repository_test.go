package store

import (
	"context"
	"testing"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/teresa-solution/tenant-management-service/internal/crypto"
	"github.com/teresa-solution/tenant-management-service/internal/model"
)

func setupTestDB(t *testing.T) (*TenantRepository, func()) {
	dsn := "postgres://admin:securepassword@localhost:5432/tenant_registry?sslmode=disable"
	repo, err := NewTenantRepository(dsn)
	assert.NoError(t, err)

	// Clear the database before each test
	_, err = repo.db.Exec("TRUNCATE TABLE tenants, tenant_schemas, tenant_provisioning_logs RESTART IDENTITY CASCADE")
	assert.NoError(t, err)

	// Clear Redis cache
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})
	rdb.FlushAll(context.Background())

	teardown := func() {
		repo.Close()
		rdb.Close()
	}

	return repo, teardown
}

func TestTenantRepository_CreateAndGet(t *testing.T) {
	repo, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()

	// Create a tenant
	tenant := &model.Tenant{
		Name:         "Test Tenant",
		Subdomain:    "test",
		ContactEmail: "test@example.com",
		Status:       "active",
		Provisioned:  true,
	}
	encryptedEmail, emailIV, err := crypto.Encrypt(tenant.ContactEmail)
	assert.NoError(t, err)
	tenant.EncryptedEmail = encryptedEmail
	tenant.EmailIV = emailIV
	err = repo.Create(ctx, tenant)
	assert.NoError(t, err)

	// Get the tenant by ID
	fetchedTenant, err := repo.GetByID(ctx, tenant.ID)
	assert.NoError(t, err)
	assert.Equal(t, tenant.ID, fetchedTenant.ID)
	assert.Equal(t, tenant.Name, fetchedTenant.Name)
	assert.Equal(t, tenant.Subdomain, fetchedTenant.Subdomain)
	assert.Equal(t, tenant.Status, fetchedTenant.Status)
	assert.Equal(t, tenant.Provisioned, fetchedTenant.Provisioned)
	assert.Equal(t, tenant.ContactEmail, fetchedTenant.ContactEmail)

	// Test cache hit
	fetchedTenant, err = repo.GetByID(ctx, tenant.ID)
	assert.NoError(t, err)
	assert.Equal(t, tenant.ID, fetchedTenant.ID)
}

func TestTenantRepository_GetBySubdomain(t *testing.T) {
	repo, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()

	// Create a tenant
	tenant := &model.Tenant{
		Name:         "Subdomain Tenant",
		Subdomain:    "subdomaintest",
		ContactEmail: "subdomain@example.com",
		Status:       "active",
		Provisioned:  true,
	}
	encryptedEmail, emailIV, err := crypto.Encrypt(tenant.ContactEmail)
	assert.NoError(t, err)
	tenant.EncryptedEmail = encryptedEmail
	tenant.EmailIV = emailIV
	err = repo.Create(ctx, tenant)
	assert.NoError(t, err)

	// Get by subdomain
	fetchedTenant, err := repo.GetBySubdomain(ctx, "subdomaintest")
	assert.NoError(t, err)
	assert.Equal(t, tenant.ID, fetchedTenant.ID)
	assert.Equal(t, tenant.Subdomain, fetchedTenant.Subdomain)

	// Test non-existent subdomain
	fetchedTenant, err = repo.GetBySubdomain(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, fetchedTenant)
}

func TestTenantRepository_Update(t *testing.T) {
	repo, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()

	// Create a tenant
	tenant := &model.Tenant{
		Name:         "Update Tenant",
		Subdomain:    "updatetest",
		ContactEmail: "update@example.com",
		Status:       "active",
		Provisioned:  true,
	}
	encryptedEmail, emailIV, err := crypto.Encrypt(tenant.ContactEmail)
	assert.NoError(t, err)
	tenant.EncryptedEmail = encryptedEmail
	tenant.EmailIV = emailIV
	err = repo.Create(ctx, tenant)
	assert.NoError(t, err)

	// Update the tenant
	tenant.Name = "Updated Tenant"
	tenant.Status = "inactive"
	err = repo.Update(ctx, tenant)
	assert.NoError(t, err)

	// Verify the update
	fetchedTenant, err := repo.GetByID(ctx, tenant.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Tenant", fetchedTenant.Name)
	assert.Equal(t, "inactive", fetchedTenant.Status)
}

func TestTenantRepository_Delete(t *testing.T) {
	repo, teardown := setupTestDB(t)
	defer teardown()

	ctx := context.Background()

	// Create a tenant
	tenant := &model.Tenant{
		Name:         "Delete Tenant",
		Subdomain:    "deletetest",
		ContactEmail: "delete@example.com",
		Status:       "active",
		Provisioned:  true,
	}
	encryptedEmail, emailIV, err := crypto.Encrypt(tenant.ContactEmail)
	assert.NoError(t, err)
	tenant.EncryptedEmail = encryptedEmail
	tenant.EmailIV = emailIV
	err = repo.Create(ctx, tenant)
	assert.NoError(t, err)

	// Delete the tenant
	err = repo.Delete(ctx, tenant.ID)
	assert.NoError(t, err)

	// Verify deletion
	fetchedTenant, err := repo.GetByID(ctx, tenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedTenant.DeletedAt)
}
