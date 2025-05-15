package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/teresa-solution/tenant-management-service/internal/model"
	"github.com/teresa-solution/tenant-management-service/internal/store"
	tenantpb "github.com/teresa-solution/tenant-management-service/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Update TenantService constructor to include ProvisioningService
type TenantService struct {
	repo                *store.TenantRepository
	provisioningService *ProvisioningService
	tenantpb.UnimplementedTenantServiceServer
}

func NewTenantService(repo *store.TenantRepository) *TenantService {
	return &TenantService{
		repo:                repo,
		provisioningService: NewProvisioningService(repo),
	}
}

// Update TenantService to integrate provisioning
func (s *TenantService) CreateTenant(ctx context.Context, req *tenantpb.CreateTenantRequest) (*tenantpb.CreateTenantResponse, error) {
	if err := validateCreateTenantRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	existingTenant, err := s.repo.GetBySubdomain(ctx, req.Subdomain)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check subdomain uniqueness")
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	if existingTenant != nil {
		return nil, status.Error(codes.AlreadyExists, "Subdomain already exists")
	}

	tenant := &model.Tenant{
		Name:      req.Name,
		Subdomain: req.Subdomain,
		Status:    "provisioning",
	}
	if err := s.repo.Create(ctx, tenant); err != nil {
		log.Error().Err(err).Msg("Failed to create tenant")
		return nil, status.Error(codes.Internal, "Failed to create tenant")
	}

	// Queue for provisioning
	if s.provisioningService != nil {
		s.provisioningService.QueueForProvisioning(tenant)
	}

	respTenant := &tenantpb.Tenant{
		Id:        tenant.ID.String(),
		Name:      tenant.Name,
		Subdomain: tenant.Subdomain,
		Status:    tenant.Status,
		CreatedAt: tenant.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: tenant.UpdatedAt.UTC().Format(time.RFC3339),
	}
	return &tenantpb.CreateTenantResponse{Tenant: respTenant}, nil
}

// GetTenant retrieves a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, req *tenantpb.GetTenantRequest) (*tenantpb.GetTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
	}

	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tenant")
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	if tenant == nil {
		return nil, status.Error(codes.NotFound, "Tenant not found")
	}

	respTenant := &tenantpb.Tenant{
		Id:        tenant.ID.String(),
		Name:      tenant.Name,
		Subdomain: tenant.Subdomain,
		Status:    tenant.Status,
		CreatedAt: tenant.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tenant.UpdatedAt.Format(time.RFC3339),
		DeletedAt: func() string {
			if tenant.DeletedAt != nil {
				return tenant.DeletedAt.Format(time.RFC3339)
			}
			return ""
		}(),
	}
	return &tenantpb.GetTenantResponse{Tenant: respTenant}, nil
}

// UpdateTenant updates an existing tenant
func (s *TenantService) UpdateTenant(ctx context.Context, req *tenantpb.UpdateTenantRequest) (*tenantpb.UpdateTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
	}

	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tenant")
		return nil, status.Error(codes.Internal, "Internal server error")
	}
	if tenant == nil {
		return nil, status.Error(codes.NotFound, "Tenant not found")
	}

	// Validate update
	if err := validateUpdateTenantRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check subdomain uniqueness if changed
	if tenant.Subdomain != req.Subdomain {
		existingTenant, err := s.repo.GetBySubdomain(ctx, req.Subdomain)
		if err != nil {
			log.Error().Err(err).Msg("Failed to check subdomain uniqueness")
			return nil, status.Error(codes.Internal, "Internal server error")
		}
		if existingTenant != nil {
			return nil, status.Error(codes.AlreadyExists, "Subdomain already exists")
		}
	}

	tenant.Name = req.Name
	tenant.Subdomain = req.Subdomain
	tenant.Status = req.Status
	if err := s.repo.Update(ctx, tenant); err != nil {
		log.Error().Err(err).Msg("Failed to update tenant")
		return nil, status.Error(codes.Internal, "Failed to update tenant")
	}

	respTenant := &tenantpb.Tenant{
		Id:        tenant.ID.String(),
		Name:      tenant.Name,
		Subdomain: tenant.Subdomain,
		Status:    tenant.Status,
		CreatedAt: tenant.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tenant.UpdatedAt.Format(time.RFC3339),
		DeletedAt: func() string {
			if tenant.DeletedAt != nil {
				return tenant.DeletedAt.Format(time.RFC3339)
			}
			return ""
		}(),
	}
	return &tenantpb.UpdateTenantResponse{Tenant: respTenant}, nil
}

// DeleteTenant soft deletes a tenant
func (s *TenantService) DeleteTenant(ctx context.Context, req *tenantpb.DeleteTenantRequest) (*tenantpb.DeleteTenantResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid tenant ID")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to delete tenant")
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &tenantpb.DeleteTenantResponse{Success: true}, nil
}

// validateCreateTenantRequest validates the create tenant request
func validateCreateTenantRequest(req *tenantpb.CreateTenantRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Subdomain == "" {
		return errors.New("subdomain is required")
	}
	if !isValidSubdomain(req.Subdomain) {
		return errors.New("invalid subdomain format")
	}
	if req.ContactEmail == "" {
		return errors.New("contact email is required")
	}
	if !isValidEmail(req.ContactEmail) {
		return errors.New("invalid email format")
	}
	return nil
}

// validateUpdateTenantRequest validates the update tenant request
func validateUpdateTenantRequest(req *tenantpb.UpdateTenantRequest) error {
	if req.Id == "" {
		return errors.New("id is required")
	}
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Subdomain == "" {
		return errors.New("subdomain is required")
	}
	if !isValidSubdomain(req.Subdomain) {
		return errors.New("invalid subdomain format")
	}
	if req.Status == "" || req.Status != "active" && req.Status != "inactive" && req.Status != "provisioning" && req.Status != "error" {
		return errors.New("invalid status")
	}
	return nil
}

// isValidSubdomain checks if the subdomain matches the regex pattern
func isValidSubdomain(subdomain string) bool {
	// Simple check based on the constraint: ^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$
	if len(subdomain) < 1 || len(subdomain) > 63 {
		return false
	}
	for i, r := range subdomain {
		if i == 0 {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') {
				return false
			}
		} else {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

// isValidEmail performs a basic email validation
func isValidEmail(email string) bool {
	// Simple check: contains @ and .
	if len(email) < 3 || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return false
	}
	return true
}
