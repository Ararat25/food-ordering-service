package domain

import "encoding/json"

// CreateOrderRequest - данные для создания заказа из корзины.
type CreateOrderRequest struct {
	VenueID         string `json:"venue_id"`         // из какого заведения заказываем
	DeliveryType    string `json:"delivery_type"`    // delivery или pickup
	DeliveryAddress string `json:"delivery_address"` // куда везти
	Comment         string `json:"comment"`          // комментарий клиента
}

// AddToCartRequest - данные для добавления блюда в корзину.
type AddToCartRequest struct {
	MenuItemID string          `json:"menu_item_id"` // какое блюдо кладём
	Quantity   int             `json:"quantity"`     // сколько штук
	Options    json.RawMessage `json:"options"`      // выбранные опции в JSON
}

// CreateReviewRequest - данные для создания отзыва.
type CreateReviewRequest struct {
	Rating  int    `json:"rating"`  // оценка от 1 до 5
	Comment string `json:"comment"` // текст отзыва
}
