package domain

import (
	"encoding/json"
	"time"
)

// Notification - уведомление для пользователя.
type Notification struct {
	ID        string          `json:"id" db:"id"`                 // уникальный id
	UserID    string          `json:"user_id" db:"user_id"`       // кому показать
	Type      string          `json:"type" db:"type"`             // тип уведомления
	Title     string          `json:"title" db:"title"`           // заголовок
	Body      string          `json:"body" db:"body"`             // текст
	Data      json.RawMessage `json:"data" db:"data"`             // доп данные в JSON
	IsRead    bool            `json:"is_read" db:"is_read"`       // прочитано или нет
	CreatedAt time.Time       `json:"created_at" db:"created_at"` // когда создано
}

// PushToken - токен устройства для пушей.
type PushToken struct {
	ID        string    `json:"id" db:"id"`                 // уникальный id
	UserID    string    `json:"user_id" db:"user_id"`       // чьё устройство
	Token     string    `json:"token" db:"token"`           // сам токен, уникален
	Platform  string    `json:"platform" db:"platform"`     // платформа, например ios или android
	CreatedAt time.Time `json:"created_at" db:"created_at"` // когда зарегистрирован
}
