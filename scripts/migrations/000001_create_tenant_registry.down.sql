-- Drop triggers and functions
DROP TRIGGER IF EXISTS trigger_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS trigger_tenant_contacts_updated_at ON tenant_contacts;
DROP TRIGGER IF EXISTS trigger_tenant_database_configs_updated_at ON tenant_database_configs;
DROP TRIGGER IF EXISTS trigger_tenant_configs_updated_at ON tenant_configs;
DROP TRIGGER IF EXISTS trigger_tenant_features_updated_at ON tenant_features;
DROP FUNCTION IF EXISTS update_updated_at;

-- Drop tables
DROP TABLE IF EXISTS tenant_audit_logs;
DROP TABLE IF EXISTS tenant_features;
DROP TABLE IF EXISTS tenant_configs;
DROP TABLE IF EXISTS tenant_database_configs;
DROP TABLE IF EXISTS tenant_contacts;
DROP TABLE IF EXISTS tenants;

DROP EXTENSION IF EXISTS "uuid-ossp";
