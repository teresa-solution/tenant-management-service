-- Create table to track tenant schema assignments
CREATE TABLE tenant_schemas (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id),
    schema_name VARCHAR(63) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Function to create a new tenant schema
CREATE OR REPLACE FUNCTION create_tenant_schema(p_tenant_id UUID, p_subdomain VARCHAR)
RETURNS VOID AS $$
BEGIN
    EXECUTE format('CREATE SCHEMA tenant_%I', p_subdomain);
    INSERT INTO tenant_schemas (tenant_id, schema_name)
    VALUES (p_tenant_id, format('tenant_%I', p_subdomain));
END;
$$ LANGUAGE plpgsql;

-- Trigger to create schema on tenant creation (optional, can be handled in code)
-- Note: We'll handle this in the service layer instead of a trigger for now
