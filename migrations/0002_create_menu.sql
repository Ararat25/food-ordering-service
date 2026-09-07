-- +goose Up
CREATE TABLE IF NOT EXISTS menu_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    venue_id UUID NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS menu_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id UUID NOT NULL REFERENCES menu_categories(id) ON DELETE CASCADE,
    venue_id UUID NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    image_url VARCHAR(500),
    is_available BOOLEAN DEFAULT true,
    options JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_menu_categories_venue_id ON menu_categories(venue_id);
CREATE INDEX idx_menu_items_venue_id ON menu_items(venue_id);
CREATE INDEX idx_menu_items_category_id ON menu_items(category_id);

-- +goose Down
DROP INDEX IF EXISTS idx_menu_items_category_id;
DROP INDEX IF EXISTS idx_menu_items_venue_id;
DROP INDEX IF EXISTS idx_menu_categories_venue_id;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS menu_categories;
