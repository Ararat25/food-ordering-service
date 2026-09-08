package domain

import "time"

// Review - отзыв пользователя о заведении.
type Review struct {
	ID        string    `json:"id" db:"id"`                 // уникальный id
	UserID    string    `json:"user_id" db:"user_id"`       // кто оставил
	VenueID   string    `json:"venue_id" db:"venue_id"`     // о каком заведении
	OrderID   string    `json:"order_id" db:"order_id"`     // по какому заказу
	Rating    int       `json:"rating" db:"rating"`         // оценка
	Comment   string    `json:"comment" db:"comment"`       // текст отзыва
	CreatedAt time.Time `json:"created_at" db:"created_at"` // когда оставлен
}
