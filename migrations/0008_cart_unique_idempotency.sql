-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_carts_user_venue_unique ON carts(user_id, venue_id);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idempotency_key ON orders(idempotency_key);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_idempotency_key;
ALTER TABLE orders DROP COLUMN IF EXISTS idempotency_key;
DROP INDEX IF EXISTS idx_carts_user_venue_unique;
