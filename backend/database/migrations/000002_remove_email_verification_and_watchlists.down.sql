BEGIN;

ALTER TABLE alert_logs DROP CONSTRAINT IF EXISTS alert_logs_channel_check;
ALTER TABLE alert_logs
    ADD CONSTRAINT alert_logs_channel_check CHECK (channel IN ('LINE', 'EMAIL'));

CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(30) NOT NULL DEFAULT 'VERIFY_EMAIL' CHECK (purpose = 'VERIFY_EMAIL'),
    otp_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (used_at IS NULL OR used_at >= created_at),
    CHECK (expires_at > created_at)
);
CREATE INDEX idx_email_verification_tokens_user_created
    ON email_verification_tokens (user_id, created_at DESC);

CREATE TABLE watchlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL DEFAULT 'My Watchlist',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_watchlists_user_name ON watchlists (user_id, LOWER(name));
CREATE INDEX idx_watchlists_user ON watchlists (user_id);
CREATE TRIGGER watchlists_set_updated_at
    BEFORE UPDATE ON watchlists
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE watchlist_stocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    watchlist_id UUID NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    stock_id UUID NOT NULL REFERENCES stocks(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (watchlist_id, stock_id)
);
CREATE INDEX idx_watchlist_stocks_stock ON watchlist_stocks (stock_id);

COMMIT;
