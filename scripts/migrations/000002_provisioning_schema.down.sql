-- Drop triggers and tables
DROP TRIGGER IF EXISTS trigger_tenant_specific_configs_updated_at ON tenant_specific_configs;
DROP TABLE IF EXISTS tenant_specific_configs;
DROP TABLE IF EXISTS tenant_provisioning_logs;
