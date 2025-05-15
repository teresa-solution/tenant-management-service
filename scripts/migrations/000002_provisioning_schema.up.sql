-- Create tenant_provisioning_logs table
CREATE TABLE IF NOT EXISTS tenant_provisioning_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    step VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'in_progress', 'success', 'failed')),
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_provisioning_logs_tenant_id ON tenant_provisioning_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_provisioning_logs_created_at ON tenant_provisioning_logs(created_at);

-- Create tenant_specific_configs table
CREATE TABLE IF NOT EXISTS tenant_specific_configs (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id),
    db_host VARCHAR(255) NOT NULL,
    db_port INTEGER NOT NULL DEFAULT 5432,
    db_name VARCHAR(63) NOT NULL,
    db_schema VARCHAR(63) NOT NULL DEFAULT 'public',
    dns_record VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TRIGGER trigger_tenant_specific_configs_updated_at
BEFORE UPDATE ON tenant_specific_configs
FOR EACH ROW EXECUTE FUNCTION update_updated_at();
