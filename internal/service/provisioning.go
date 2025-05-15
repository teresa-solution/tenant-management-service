package service

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/teresa-solution/tenant-management-service/internal/model"
	"github.com/teresa-solution/tenant-management-service/internal/store"
)

// ProvisioningService handles tenant provisioning workflows
type ProvisioningService struct {
	repo         *store.TenantRepository
	provisioning chan *model.Tenant // Channel for background provisioning
}

// NewProvisioningService creates a new ProvisioningService
func NewProvisioningService(repo *store.TenantRepository) *ProvisioningService {
	ps := &ProvisioningService{
		repo:         repo,
		provisioning: make(chan *model.Tenant, 10),
	}
	go ps.startProvisioningWorker()
	return ps
}

// startProvisioningWorker runs the background job for provisioning
func (ps *ProvisioningService) startProvisioningWorker() {
	for tenant := range ps.provisioning {
		log.Info().Str("tenant_id", tenant.ID.String()).Msg("Starting provisioning process")
		if err := ps.provisionTenant(tenant); err != nil {
			log.Error().Err(err).Str("tenant_id", tenant.ID.String()).Msg("Provisioning failed")
		}
	}
}

// provisionTenant simulates the provisioning process
func (ps *ProvisioningService) provisionTenant(tenant *model.Tenant) error {
	ctx := context.Background()

	// Log provisioning start
	if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "init", "pending", nil); err != nil {
		return err
	}

	// Simulate provisioning steps (e.g., database setup, DNS registration)
	time.Sleep(2 * time.Second) // Simulate external system call
	if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "db_setup", "in_progress", map[string]interface{}{"host": "db.example.com"}); err != nil {
		return err
	}

	// Simulate success or failure
	if time.Now().UnixNano()%2 == 0 { // Random success/failure for demo
		if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "db_setup", "success", nil); err != nil {
			return err
		}
		tenant.Status = "active"
		tenant.Provisioned = true
	} else {
		if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "db_setup", "failed", map[string]interface{}{"error": "timeout"}); err != nil {
			return err
		}
		tenant.Status = "error"
		return errors.New("provisioning failed")
	}

	if err := ps.repo.Update(ctx, tenant); err != nil {
		return err
	}

	return nil
}

// QueueForProvisioning adds a tenant to the provisioning queue
func (ps *ProvisioningService) QueueForProvisioning(tenant *model.Tenant) {
	ps.provisioning <- tenant
}
