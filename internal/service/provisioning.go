package service

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/teresa-solution/tenant-management-service/internal/model"
	"github.com/teresa-solution/tenant-management-service/internal/monitoring"
	"github.com/teresa-solution/tenant-management-service/internal/store"
)

// ProvisioningServiceInterface defines the methods required for provisioning
type ProvisioningServiceInterface interface {
	QueueForProvisioning(tenant *model.Tenant)
}



// ProvisioningService handles tenant provisioning workflows
type ProvisioningService struct {
	repo         *store.TenantRepository
	provisioning chan *model.Tenant
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
		log.Info().
			Str("tenant_id", tenant.ID.String()).
			Str("subdomain", tenant.Subdomain).
			Msg("Starting provisioning process")
		if err := ps.provisionTenant(tenant); err != nil {
			log.Error().
				Str("tenant_id", tenant.ID.String()).
				Err(err).
				Msg("Provisioning failed")
		}
	}
}

// provisionTenant simulates the provisioning process
func (ps *ProvisioningService) provisionTenant(tenant *model.Tenant) error {
	startTime := time.Now()
	ctx := context.Background()

	log.Info().
		Str("tenant_id", tenant.ID.String()).
		Str("subdomain", tenant.Subdomain).
		Msg("Starting provisioning process")

	// Create tenant schema
	if err := ps.repo.CreateTenantSchema(ctx, tenant.ID, tenant.Subdomain); err != nil {
		log.Error().
			Str("tenant_id", tenant.ID.String()).
			Err(err).
			Msg("Failed to create tenant schema")
		return err
	}

	if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "init", "pending", nil); err != nil {
		log.Error().
			Str("tenant_id", tenant.ID.String()).
			Err(err).
			Msg("Failed to create provisioning log")
		return err
	}

	time.Sleep(2 * time.Second)
	if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "db_setup", "in_progress", map[string]interface{}{"host": "db.example.com"}); err != nil {
		log.Error().
			Str("tenant_id", tenant.ID.String()).
			Err(err).
			Msg("Failed to log db_setup step")
		return err
	}

	var provisioningStatus string
	if time.Now().UnixNano()%2 == 0 {
		if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "db_setup", "success", nil); err != nil {
			log.Error().
				Str("tenant_id", tenant.ID.String()).
				Err(err).
				Msg("Failed to log db_setup success")
			return err
		}
		tenant.Status = "active"
		tenant.Provisioned = true
		provisioningStatus = "success"
		log.Info().
			Str("tenant_id", tenant.ID.String()).
			Msg("Provisioning completed successfully")
	} else {
		if err := ps.repo.CreateProvisioningLog(ctx, tenant.ID, "db_setup", "failed", map[string]interface{}{"error": "timeout"}); err != nil {
			log.Error().
				Str("tenant_id", tenant.ID.String()).
				Err(err).
				Msg("Failed to log db_setup failure")
			return err
		}
		tenant.Status = "error"
		provisioningStatus = "failed"
		log.Warn().
			Str("tenant_id", tenant.ID.String()).
			Msg("Provisioning failed due to timeout")

		monitoring.MockAlert("Tenant provisioning failed", map[string]string{
			"tenant_id": tenant.ID.String(),
			"subdomain": tenant.Subdomain,
			"error":     "timeout",
		})
	}

	monitoring.TenantsProvisioned.WithLabelValues(provisioningStatus).Inc()
	duration := time.Since(startTime).Seconds()
	monitoring.ProvisioningDuration.Observe(duration)

	if err := ps.repo.Update(ctx, tenant); err != nil {
		log.Error().
			Str("tenant_id", tenant.ID.String()).
			Err(err).
			Msg("Failed to update tenant status after provisioning")
		return err
	}

	return nil
}

// QueueForProvisioning adds a tenant to the provisioning queue
func (ps *ProvisioningService) QueueForProvisioning(tenant *model.Tenant) {
	ps.provisioning <- tenant
}
