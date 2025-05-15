CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    subdomain VARCHAR(63) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'inactive', 'provisioning', 'error')),
    provisioned BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT subdomain_format CHECK (subdomain ~ '^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$')
);

CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_subdomain ON tenants(subdomain) WHERE deleted_at IS NULL;

-- Create tenant_contacts table
CREATE TABLE IF NOT EXISTS tenant_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('primary', 'billing', 'technical', 'emergency')),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, type)
);

CREATE INDEX IF NOT EXISTS idx_tenant_contacts_tenant_id ON tenant_contacts(tenant_id);

-- Create tenant_database_configs table
CREATE TABLE IF NOT EXISTS tenant_database_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id),
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 5432,
    database_name VARCHAR(63) NOT NULL,
    schema_name VARCHAR(63) NOT NULL DEFAULT 'public',
    username VARCHAR(63) NOT NULL,
    password_secret_id VARCHAR(255) NOT NULL,
    max_connections INTEGER NOT NULL DEFAULT 10,
    idle_connections INTEGER NOT NULL DEFAULT 2,
    connection_lifetime_minutes INTEGER NOT NULL DEFAULT 60,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_database_configs_tenant_id ON tenant_database_configs(tenant_id);

-- Create tenant_configs table
CREATE TABLE IF NOT EXISTS tenant_configs (
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    key VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_tenant_configs_tenant_id ON tenant_configs(tenant_id);

-- Create tenant_features table
CREATE TABLE IF NOT EXISTS tenant_features (
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    feature_name VARCHAR(50) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, feature_name)
);

CREATE INDEX IF NOT EXISTS idx_tenant_features_tenant_id ON tenant_features(tenant_id);

-- Create tenant_audit_logs table
CREATE TABLE IF NOT EXISTS tenant_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    action VARCHAR(50) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    details JSONB NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_tenant_id ON tenant_audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_created_at ON tenant_audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_tenant_audit_logs_action ON tenant_audit_logs(action);

-- Create update_updated_at function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update_updated_at trigger to relevant tables
CREATE TRIGGER trigger_tenants_updated_at
BEFORE UPDATE ON tenants
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_tenant_contacts_updated_at
BEFORE UPDATE ON tenant_contacts
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_tenant_database_configs_updated_at
BEFORE UPDATE ON tenant_database_configs
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_tenant_configs_updated_at
BEFORE UPDATE ON tenant_configs
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trigger_tenant_features_updated_at
BEFORE UPDATE ON tenant_features
FOR EACH ROW EXECUTE FUNCTION update_updated_at();
