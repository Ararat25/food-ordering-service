package domain

import "time"

// User - пользователь сервиса.
type User struct {
	ID        string    `json:"id" db:"id"`                 // уникальный id
	Name      string    `json:"name" db:"name"`             // имя
	Email     string    `json:"email" db:"email"`           // email, уникален
	Phone     string    `json:"phone" db:"phone"`           // телефон
	CreatedAt time.Time `json:"created_at" db:"created_at"` // когда создан
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"` // когда обновлён
}
