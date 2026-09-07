package domain

import (
	"encoding/json"
	"time"
)

// Venue - заведение (ресторан, кафе, магазин).
type Venue struct {
	ID           string          `json:"id" db:"id"`                       // уникальный id
	Name         string          `json:"name" db:"name"`                   // название
	Description  string          `json:"description" db:"description"`     // описание
	LogoURL      string          `json:"logo_url" db:"logo_url"`           // ссылка на логотип
	Phone        string          `json:"phone" db:"phone"`                 // телефон
	Address      string          `json:"address" db:"address"`             // адрес
	Latitude     float64         `json:"latitude" db:"latitude"`           // широта
	Longitude    float64         `json:"longitude" db:"longitude"`         // долгота
	WorkingHours json.RawMessage `json:"working_hours" db:"working_hours"` // часы работы в JSON
	DeliveryZone json.RawMessage `json:"delivery_zone" db:"delivery_zone"` // зона доставки в JSON
	IsActive     bool            `json:"is_active" db:"is_active"`         // принимает ли заказы
	APIKey       string          `json:"-" db:"api_key"`                   // ключ для API заведения, наружу не отдаём
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`       // когда создано
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`       // когда обновлено
}
