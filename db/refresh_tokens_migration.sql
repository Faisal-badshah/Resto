CREATE TABLE IF NOT EXISTS refresh_tokens (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id),
  admin_email TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  expires_at TIMESTAMP WITH TIME ZONE,
  revoked BOOLEAN DEFAULT FALSE,
  ip TEXT,
  user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_email ON refresh_tokens(admin_email);
