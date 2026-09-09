BEGIN;

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name VARCHAR(150),
    email CITEXT UNIQUE,
    password_hash TEXT,
    email_verified_at TIMESTAMPTZ,
    role VARCHAR(20) NOT NULL DEFAULT 'USER' CHECK (role IN ('USER', 'ADMIN')),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    disabled_reason TEXT,
    disabled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((status = 'ACTIVE' AND disabled_at IS NULL) OR status = 'DISABLED')
);

CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL CHECK (provider IN ('LINE')),
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider)
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    replaced_by_token_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_ip INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at > created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE TABLE stocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) NOT NULL,
    provider VARCHAR(30) NOT NULL DEFAULT 'FINNHUB',
    provider_symbol VARCHAR(100),
    name VARCHAR(255),
    exchange VARCHAR(50),
    currency VARCHAR(10),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    disabled_reason TEXT,
    disabled_at TIMESTAMPTZ,
    last_quote_price NUMERIC(20, 8),
    last_quote_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, symbol),
    CHECK (symbol = UPPER(symbol)),
    CHECK (last_quote_price IS NULL OR last_quote_price >= 0),
    CHECK ((status = 'ACTIVE' AND disabled_at IS NULL) OR status = 'DISABLED')
);

CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stock_id UUID NOT NULL REFERENCES stocks(id) ON DELETE RESTRICT,
    condition VARCHAR(10) NOT NULL CHECK (condition IN ('ABOVE', 'BELOW')),
    target_price NUMERIC(20, 8) NOT NULL CHECK (target_price > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'TRIGGERED', 'DISABLED')),
    triggered_at TIMESTAMPTZ,
    disabled_reason TEXT,
    disabled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((status = 'TRIGGERED' AND triggered_at IS NOT NULL) OR (status <> 'TRIGGERED' AND triggered_at IS NULL)),
    CHECK ((status = 'DISABLED' AND disabled_at IS NOT NULL) OR (status <> 'DISABLED' AND disabled_at IS NULL))
);
CREATE UNIQUE INDEX uq_alerts_active_definition
    ON alerts (user_id, stock_id, condition, target_price) WHERE status = 'ACTIVE';

CREATE TABLE alert_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel = 'LINE'),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'SENT', 'FAILED')),
    message TEXT,
    triggered_price NUMERIC(20, 8) NOT NULL CHECK (triggered_price >= 0),
    provider_message_id VARCHAR(255),
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((status = 'SENT' AND sent_at IS NOT NULL) OR status <> 'SENT'),
    CHECK ((status = 'FAILED' AND error_message IS NOT NULL) OR status <> 'FAILED')
);

CREATE TABLE line_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_event_id VARCHAR(255) NOT NULL UNIQUE,
    line_user_id VARCHAR(255),
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED' CHECK (status IN ('RECEIVED', 'PROCESSED', 'FAILED')),
    processed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((status = 'PROCESSED' AND processed_at IS NOT NULL) OR status <> 'PROCESSED'),
    CHECK ((status = 'FAILED' AND error_message IS NOT NULL) OR status <> 'FAILED')
);

CREATE TABLE application_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level VARCHAR(20) NOT NULL CHECK (level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    service VARCHAR(100) NOT NULL,
    code VARCHAR(100),
    message TEXT NOT NULL,
    details JSONB,
    request_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE admin_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id TEXT,
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_identities_user_id ON user_identities (user_id);
CREATE INDEX idx_refresh_tokens_user_active ON refresh_tokens (user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens (family_id);
CREATE INDEX idx_stocks_symbol ON stocks (symbol);
CREATE INDEX idx_alerts_active_stock ON alerts (stock_id) WHERE status = 'ACTIVE';
CREATE INDEX idx_alerts_user_created ON alerts (user_id, created_at DESC);
CREATE INDEX idx_alert_logs_alert_created ON alert_logs (alert_id, created_at DESC);
CREATE INDEX idx_alert_logs_status_created ON alert_logs (status, created_at DESC);
CREATE INDEX idx_line_webhook_events_status_created ON line_webhook_events (status, created_at);
CREATE INDEX idx_application_logs_filters ON application_logs (level, service, created_at DESC);
CREATE INDEX idx_admin_audit_logs_created ON admin_audit_logs (created_at DESC);
CREATE INDEX idx_admin_audit_logs_admin_created ON admin_audit_logs (admin_user_id, created_at DESC);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER user_identities_set_updated_at BEFORE UPDATE ON user_identities FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER stocks_set_updated_at BEFORE UPDATE ON stocks FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER alerts_set_updated_at BEFORE UPDATE ON alerts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
