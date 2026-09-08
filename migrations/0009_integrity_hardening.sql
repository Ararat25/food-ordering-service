-- +goose Up
DROP INDEX IF EXISTS idx_orders_idempotency_key;
CREATE UNIQUE INDEX idx_orders_user_idempotency_key
    ON orders(user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

ALTER TABLE menu_categories
    ADD CONSTRAINT menu_categories_id_venue_unique UNIQUE (id, venue_id);
ALTER TABLE menu_items
    ADD CONSTRAINT menu_items_category_venue_fk
    FOREIGN KEY (category_id, venue_id)
    REFERENCES menu_categories(id, venue_id)
    ON DELETE CASCADE NOT VALID;

ALTER TABLE cart_items
    ADD CONSTRAINT cart_items_quantity_range CHECK (quantity BETWEEN 1 AND 1000) NOT VALID;
ALTER TABLE order_items
    ADD CONSTRAINT order_items_quantity_range CHECK (quantity BETWEEN 1 AND 1000) NOT VALID;
ALTER TABLE menu_items
    ADD CONSTRAINT menu_items_price_nonnegative CHECK (price >= 0) NOT VALID;
ALTER TABLE order_items
    ADD CONSTRAINT order_items_price_nonnegative CHECK (price >= 0) NOT VALID;
ALTER TABLE orders
    ADD CONSTRAINT orders_total_nonnegative CHECK (total_amount >= 0) NOT VALID;
ALTER TABLE payments
    ADD CONSTRAINT payments_amount_nonnegative CHECK (amount >= 0) NOT VALID;

INSERT INTO users (id, name, email, phone) VALUES
    ('550e8400-e29b-41d4-a716-446655440002', 'Тестовый пользователь 2', 'user2@example.com', '+7 999 123 4568')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM users u
WHERE u.id = '550e8400-e29b-41d4-a716-446655440002'
  AND NOT EXISTS (SELECT 1 FROM carts c WHERE c.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM reviews r WHERE r.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM notifications n WHERE n.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM push_tokens p WHERE p.user_id = u.id);
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_amount_nonnegative;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_total_nonnegative;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_price_nonnegative;
ALTER TABLE menu_items DROP CONSTRAINT IF EXISTS menu_items_price_nonnegative;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_quantity_range;
ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_quantity_range;
ALTER TABLE menu_items DROP CONSTRAINT IF EXISTS menu_items_category_venue_fk;
ALTER TABLE menu_categories DROP CONSTRAINT IF EXISTS menu_categories_id_venue_unique;
DROP INDEX IF EXISTS idx_orders_user_idempotency_key;

CREATE UNIQUE INDEX idx_orders_idempotency_key ON orders(idempotency_key);
