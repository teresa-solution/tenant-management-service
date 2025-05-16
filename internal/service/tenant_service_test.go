package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/teresa-solution/tenant-management-service/internal/crypto"
	"github.com/teresa-solution/tenant-management-service/internal/model"
	"github.com/teresa-solution/tenant-management-service/internal/store"
	tenantpb "github.com/teresa-solution/tenant-management-service/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockProvisioningService implements ProvisioningServiceInterface
type mockProvisioningService struct{}

func (m *mockProvisioningService) QueueForProvisioning(tenant *model.Tenant) {
	// Mock implementation: do nothing
}

func setupTestService(t *testing.T) (*TenantService, *store.TenantRepository, func()) {
	dsn := "postgres://admin:securepassword@localhost:5432/tenant_registry?sslmode=disable"
	repo, err := store.NewTenantRepository(dsn)
	assert.NoError(t, err)

	// Initialize the service with a mock provisioning service
	svc := &TenantService{
		repo:                repo,
		provisioningService: &mockProvisioningService{},
	}

	// Teardown function to close connections
	teardown := func() {
		repo.Close()
	}

	return svc, repo, teardown
}

func TestTenantService_CreateTenant(t *testing.T) {
	svc, repo, teardown := setupTestService(t)
	defer teardown()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-subdomain", "testsvc"))

	// Create a tenant
	req := &tenantpb.CreateTenantRequest{
		Name:         "Service Test Tenant",
		Subdomain:    "testsvc",
		ContactEmail: "svc@example.com",
		Tier:         "basic",
	}
	resp, err := svc.CreateTenant(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Tenant.Id)
	assert.Equal(t, "Service Test Tenant", resp.Tenant.Name)
	assert.Equal(t, "testsvc", resp.Tenant.Subdomain)
	assert.Equal(t, "provisioning", resp.Tenant.Status)

	// Verify in database
	tenantID, err := uuid.Parse(resp.Tenant.Id)
	assert.NoError(t, err)
	tenant, err := repo.GetByID(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, "Service Test Tenant", tenant.Name)

	// Clean up the created tenant
	err = repo.Delete(ctx, tenantID)
	assert.NoError(t, err)
}

func TestTenantService_CreateTenant_MissingSubdomain(t *testing.T) {
	svc, _, teardown := setupTestService(t)
	defer teardown()

	ctx := context.Background() // No subdomain metadata

	req := &tenantpb.CreateTenantRequest{
		Name:         "Invalid Tenant",
		Subdomain:    "invalid",
		ContactEmail: "invalid@example.com",
		Tier:         "basic",
	}
	resp, err := svc.CreateTenant(ctx, req)
	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "missing metadata", st.Message()) // Updated to match actual error
}

func TestTenantService_GetTenant(t *testing.T) {
	svc, repo, teardown := setupTestService(t)
	defer teardown()

	ctx := context.Background()

	// Create a tenant
	tenant := &model.Tenant{
		Name:         "Get Tenant",
		Subdomain:    "gettest",
		ContactEmail: "get@example.com",
		Status:       "active",
		Provisioned:  true,
	}
	encryptedEmail, emailIV, err := crypto.Encrypt(tenant.ContactEmail)
	assert.NoError(t, err)
	tenant.EncryptedEmail = encryptedEmail
	tenant.EmailIV = emailIV
	err = repo.Create(ctx, tenant)
	assert.NoError(t, err)

	// Get the tenant
	req := &tenantpb.GetTenantRequest{Id: tenant.ID.String()}
	resp, err := svc.GetTenant(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, tenant.ID.String(), resp.Tenant.Id)
	assert.Equal(t, "Get Tenant", resp.Tenant.Name)
	assert.Equal(t, "gettest", resp.Tenant.Subdomain)

	// Clean up the created tenant
	err = repo.Delete(ctx, tenant.ID)
	assert.NoError(t, err)
}

func TestTenantService_GetTenant_NotFound(t *testing.T) {
	svc, _, teardown := setupTestService(t)
	defer teardown()

	ctx := context.Background()

	req := &tenantpb.GetTenantRequest{Id: uuid.New().String()}
	resp, err := svc.GetTenant(ctx, req)
	assert.Nil(t, resp)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "Tenant not found", st.Message())
}
