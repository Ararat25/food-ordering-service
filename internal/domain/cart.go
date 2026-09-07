package domain

import (
	"encoding/json"
	"time"
)

// Cart - корзина пользователя в одном заведении.
type Cart struct {
	ID        string     `json:"id" db:"id"`                 // уникальный id
	UserID    string     `json:"user_id" db:"user_id"`       // чей пользователь
	VenueID   string     `json:"venue_id" db:"venue_id"`     // чьё заведение
	Items     []CartItem `json:"items"`                      // позиции в корзине, подгружаются отдельно
	CreatedAt time.Time  `json:"created_at" db:"created_at"` // когда создана
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"` // когда обновлена
}

// CartItem - одна позиция в корзине.
type CartItem struct {
	ID         string          `json:"id" db:"id"`                     // уникальный id
	CartID     string          `json:"cart_id" db:"cart_id"`           // чья корзина
	MenuItemID string          `json:"menu_item_id" db:"menu_item_id"` // какое блюдо
	Quantity   int             `json:"quantity" db:"quantity"`         // количество
	Options    json.RawMessage `json:"options" db:"options"`           // выбранные опции в JSON
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`     // когда добавлено
}
