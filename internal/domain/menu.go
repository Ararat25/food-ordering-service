package domain

import (
	"encoding/json"
	"time"
)

// MenuCategory - категория блюд в меню заведения.
type MenuCategory struct {
	ID        string    `json:"id" db:"id"`                 // уникальный id
	VenueID   string    `json:"venue_id" db:"venue_id"`     // чьё заведение
	Name      string    `json:"name" db:"name"`             // название категории
	SortOrder int       `json:"sort_order" db:"sort_order"` // порядок показа
	CreatedAt time.Time `json:"created_at" db:"created_at"` // когда создана
}

// MenuItem - блюдо в меню заведения.
type MenuItem struct {
	ID          string          `json:"id" db:"id"`                     // уникальный id
	CategoryID  string          `json:"category_id" db:"category_id"`   // категория блюда
	VenueID     string          `json:"venue_id" db:"venue_id"`         // чьё заведение
	Name        string          `json:"name" db:"name"`                 // название
	Description string          `json:"description" db:"description"`   // описание
	Price       int64           `json:"price" db:"price"`               // цена в копейках
	ImageURL    string          `json:"image_url" db:"image_url"`       // ссылка на фото
	IsAvailable bool            `json:"is_available" db:"is_available"` // есть ли в наличии
	Options     json.RawMessage `json:"options" db:"options"`           // опции блюда в JSON
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`     // когда создано
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`     // когда обновлено
}
