BEGIN;

DROP TABLE IF EXISTS watchlist_stocks;
DROP TABLE IF EXISTS watchlists;
DROP TABLE IF EXISTS email_verification_tokens;

ALTER TABLE alert_logs DROP CONSTRAINT IF EXISTS alert_logs_channel_check;
ALTER TABLE alert_logs
    ADD CONSTRAINT alert_logs_channel_check CHECK (channel = 'LINE');

COMMIT;
