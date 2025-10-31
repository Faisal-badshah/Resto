CREATE TABLE IF NOT EXISTS password_resets (
  id SERIAL PRIMARY KEY,
  admin_email TEXT NOT NULL,
  token TEXT NOT NULL UNIQUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  expires_at TIMESTAMP WITH TIME ZONE,
  used_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_password_resets_token ON password_resets(token);
CREATE INDEX IF NOT EXISTS idx_password_resets_email ON password_resets(admin_email);
