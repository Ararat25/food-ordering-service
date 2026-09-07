-- +goose Up
INSERT INTO users (id, name, email, phone) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'Тестовый пользователь', 'user@example.com', '+7 999 123 4567')
ON CONFLICT (id) DO NOTHING;

INSERT INTO venues (id, name, description, logo_url, phone, address, latitude, longitude, working_hours, delivery_zone, is_active, api_key) VALUES
('660e8400-e29b-41d4-a716-446655440001', 'Пиццерия "Итальянец"', 'Лучшая пицца в городе', '/images/pizza.png', '+7 999 222 3344', 'ул. Пушкина, д. 10', 55.7558, 37.6173, '{"monday":"10:00-22:00","tuesday":"10:00-22:00","wednesday":"10:00-22:00","thursday":"10:00-22:00","friday":"10:00-23:00","saturday":"11:00-23:00","sunday":"11:00-22:00"}', '{"type":"polygon","coordinates":[[55.75,37.61],[55.76,37.61],[55.76,37.62],[55.75,37.62]]}', true, 'vk_demo_key_for_venue_1234567890abcdef')
ON CONFLICT (id) DO NOTHING;

INSERT INTO venues (id, name, description, logo_url, phone, address, latitude, longitude, working_hours, delivery_zone, is_active, api_key) VALUES
('660e8400-e29b-41d4-a716-446655440002', 'Суши "Восток"', 'Японская кухня', '/images/sushi.png', '+7 999 333 4455', 'ул. Ленина, д. 25', 55.7600, 37.6200, '{"monday":"11:00-23:00","tuesday":"11:00-23:00","wednesday":"11:00-23:00","thursday":"11:00-23:00","friday":"11:00-00:00","saturday":"11:00-00:00","sunday":"11:00-23:00"}', '{"type":"circle","center":[55.76,37.62],"radius":3000}', true, 'vk_demo_key_for_venue_1234567890abcdef2')
ON CONFLICT (id) DO NOTHING;

INSERT INTO menu_categories (id, venue_id, name, sort_order) VALUES
('770e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'Пицца', 1),
('770e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440001', 'Напитки', 2),
('770e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440002', 'Роллы', 1),
('770e8400-e29b-41d4-a716-446655440004', '660e8400-e29b-41d4-a716-446655440002', 'Супы', 2)
ON CONFLICT (id) DO NOTHING;

INSERT INTO menu_items (id, category_id, venue_id, name, description, price, image_url, is_available, options) VALUES
('880e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'Маргарита', 'Классическая пицца с моцареллой', 590.00, '/images/margarita.jpg', true, '[{"name":"size","options":["small","medium","large"],"default":"medium"}]'),
('880e8400-e29b-41d4-a716-446655440002', '770e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'Пепперони', 'Пицца с пепперони и сыром', 690.00, '/images/pepperoni.jpg', true, '[{"name":"size","options":["small","medium","large"],"default":"medium"}]'),
('880e8400-e29b-41d4-a716-446655440003', '770e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440001', 'Кола', 'Coca-Cola 0.5л', 150.00, '/images/cola.jpg', true, '[]'),
('880e8400-e29b-41d4-a716-446655440004', '770e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440002', 'Филадельфия', 'Лосось, сливочный сыр, огурец', 450.00, '/images/philadelphia.jpg', true, '[{"name":"count","options":["6","8","10"],"default":"8"}]'),
('880e8400-e29b-41d4-a716-446655440005', '770e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440002', 'Калифорния', 'Краб, авокадо, огурец, икра тобико', 520.00, '/images/california.jpg', true, '[{"name":"count","options":["6","8","10"],"default":"8"}]'),
('880e8400-e29b-41d4-a716-446655440006', '770e8400-e29b-41d4-a716-446655440004', '660e8400-e29b-41d4-a716-446655440002', 'Мисо суп', 'Традиционный японский суп', 280.00, '/images/miso.jpg', true, '[]')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM menu_items WHERE id IN ('880e8400-e29b-41d4-a716-446655440001','880e8400-e29b-41d4-a716-446655440002','880e8400-e29b-41d4-a716-446655440003','880e8400-e29b-41d4-a716-446655440004','880e8400-e29b-41d4-a716-446655440005','880e8400-e29b-41d4-a716-446655440006');
DELETE FROM menu_categories WHERE id IN ('770e8400-e29b-41d4-a716-446655440001','770e8400-e29b-41d4-a716-446655440002','770e8400-e29b-41d4-a716-446655440003','770e8400-e29b-41d4-a716-446655440004');
DELETE FROM venues WHERE id IN ('660e8400-e29b-41d4-a716-446655440001','660e8400-e29b-41d4-a716-446655440002');
DELETE FROM users WHERE id = '550e8400-e29b-41d4-a716-446655440001';