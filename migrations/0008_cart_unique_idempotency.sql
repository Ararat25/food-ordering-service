-- +goose Up
-- Одна корзина на пару пользователь-заведение: без индекса два параллельных
-- AddItem создавали две корзины, заказ потом собирался только из одной.
CREATE UNIQUE INDEX IF NOT EXISTS idx_carts_user_venue_unique ON carts(user_id, venue_id);

-- Ключ идемпотентности: повторный POST /orders с тем же ключом возвращает
-- уже созданный заказ вместо дубля. NULL-ключи друг с другом не конфликтуют.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idempotency_key ON orders(idempotency_key);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_idempotency_key;
ALTER TABLE orders DROP COLUMN IF EXISTS idempotency_key;
DROP INDEX IF EXISTS idx_carts_user_venue_unique;
