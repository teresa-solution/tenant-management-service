ALTER TABLE tenants ADD COLUMN encrypted_email BYTEA;
ALTER TABLE tenants ADD COLUMN email_iv BYTEA;
