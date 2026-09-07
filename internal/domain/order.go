package domain

import (
	"encoding/json"
	"time"
)

// OrderStatus - статус заказа.
type OrderStatus string

// Возможные статусы заказа по порядку жизни.
const (
	OrderStatusPending    OrderStatus = "pending"    // создан, ждёт заведения
	OrderStatusAccepted   OrderStatus = "accepted"   // заведение приняло
	OrderStatusPreparing  OrderStatus = "preparing"  // готовится
	OrderStatusReady      OrderStatus = "ready"      // готов
	OrderStatusDelivering OrderStatus = "delivering" // едет к клиенту
	OrderStatusDelivered  OrderStatus = "delivered"  // доставлен
	OrderStatusCancelled  OrderStatus = "cancelled"  // отменён
)

// DeliveryType - способ получения заказа.
type DeliveryType string

// Возможные способы получения.
const (
	DeliveryTypeDelivery DeliveryType = "delivery" // доставка
	DeliveryTypePickup   DeliveryType = "pickup"   // самовывоз
)

// Order - заказ пользователя.
type Order struct {
	ID                    string       `json:"id" db:"id"`                                           // уникальный id
	UserID                string       `json:"user_id" db:"user_id"`                                 // чей пользователь
	VenueID               string       `json:"venue_id" db:"venue_id"`                               // чьё заведение
	Status                OrderStatus  `json:"status" db:"status"`                                   // текущий статус
	DeliveryType          DeliveryType `json:"delivery_type" db:"delivery_type"`                     // доставка или самовывоз
	DeliveryAddress       string       `json:"delivery_address" db:"delivery_address"`               // куда везти
	DeliveryTime          *time.Time   `json:"delivery_time" db:"delivery_time"`                     // когда доставили
	EstimatedDeliveryTime *time.Time   `json:"estimated_delivery_time" db:"estimated_delivery_time"` // когда обещаем доставить
	Comment               string       `json:"comment" db:"comment"`                                 // комментарий клиента
	TotalAmount           float64      `json:"total_amount" db:"total_amount"`                       // итоговая сумма
	Items                 []OrderItem  `json:"items"`                                                // позиции заказа, подгружаются отдельно
	CreatedAt             time.Time    `json:"created_at" db:"created_at"`                           // когда создан
	UpdatedAt             time.Time    `json:"updated_at" db:"updated_at"`                           // когда обновлён
}

// OrderItem - одна позиция в заказе.
type OrderItem struct {
	ID         string          `json:"id" db:"id"`                     // уникальный id
	OrderID    string          `json:"order_id" db:"order_id"`         // чей заказ
	MenuItemID string          `json:"menu_item_id" db:"menu_item_id"` // какое блюдо
	Quantity   int             `json:"quantity" db:"quantity"`         // количество
	Price      float64         `json:"price" db:"price"`               // цена на момент заказа
	Options    json.RawMessage `json:"options" db:"options"`           // выбранные опции в JSON
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`     // когда добавлено
}

// PaymentStatus - статус оплаты.
type PaymentStatus string

// Возможные статусы оплаты.
const (
	PaymentStatusPending   PaymentStatus = "pending"   // ждёт оплаты
	PaymentStatusCompleted PaymentStatus = "completed" // оплачен
	PaymentStatusFailed    PaymentStatus = "failed"    // не прошёл
	PaymentStatusRefunded  PaymentStatus = "refunded"  // возвращён
)

// Payment - оплата заказа.
type Payment struct {
	ID        string        `json:"id" db:"id"`                 // уникальный id
	OrderID   string        `json:"order_id" db:"order_id"`     // чей заказ
	Status    PaymentStatus `json:"status" db:"status"`         // текущий статус
	Amount    float64       `json:"amount" db:"amount"`         // сумма
	Method    string        `json:"method" db:"method"`         // способ оплаты
	CreatedAt time.Time     `json:"created_at" db:"created_at"` // когда создан
}
