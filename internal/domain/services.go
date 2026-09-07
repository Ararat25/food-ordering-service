package domain

import "context"

// VenueService - логика заведений.
type VenueService interface {
	List(ctx context.Context, filters VenueFilters) ([]Venue, int, error) // список с фильтрами
	GetByID(ctx context.Context, id string) (*Venue, error)               // одно по id
	GetByAPIKey(ctx context.Context, apiKey string) (*Venue, error)       // одно по API-ключу, для auth заведений
	Create(ctx context.Context, venue *Venue) error                       // создать
}

// MenuService - логика меню.
type MenuService interface {
	ListCategories(ctx context.Context, venueID string) ([]MenuCategory, error)       // категории заведения
	CreateCategory(ctx context.Context, category *MenuCategory) error                 // создать категорию
	UpdateCategory(ctx context.Context, category *MenuCategory) error                 // обновить категорию
	DeleteCategory(ctx context.Context, id string) error                              // удалить категорию
	ListItems(ctx context.Context, venueID string) ([]MenuItem, error)                // блюда заведения
	GetItem(ctx context.Context, id string) (*MenuItem, error)                        // одно блюдо по id
	CreateItem(ctx context.Context, item *MenuItem) error                             // создать блюдо
	UpdateItem(ctx context.Context, item *MenuItem) error                             // обновить блюдо
	DeleteItem(ctx context.Context, venueID, id string) error                         // удалить блюдо своего заведения
	UpdateAvailability(ctx context.Context, venueID, id string, available bool) error // сменить наличие у своего блюда
}

// CartService - логика корзины.
type CartService interface {
	GetCart(ctx context.Context, userID, venueID string) (*Cart, error)                       // корзина с позициями
	AddItem(ctx context.Context, userID, venueID string, req AddToCartRequest) (*Cart, error) // добавить блюдо
	UpdateItemQuantity(ctx context.Context, userID, itemID string, quantity int) error        // сменить количество
	RemoveItem(ctx context.Context, userID, itemID string) error                              // убрать позицию
	Clear(ctx context.Context, userID, venueID string) error                                  // очистить корзину
}

// OrderService - логика заказов.
type OrderService interface {
	Create(ctx context.Context, userID string, req CreateOrderRequest) (*Order, error)                       // создать заказ из корзины
	GetByID(ctx context.Context, id string) (*Order, error)                                                  // один по id
	ListByUser(ctx context.Context, userID string, page, perPage int) ([]Order, int, error)                  // заказы пользователя
	ListByVenue(ctx context.Context, venueID string, status string, page, perPage int) ([]Order, int, error) // заказы заведения
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error                                   // сменить статус
	UpdateVenueOrderStatus(ctx context.Context, venueID, id string, status OrderStatus) error                // сменить статус только своего заказа
	GetItems(ctx context.Context, orderID string) ([]OrderItem, error)                                       // позиции заказа
	Repeat(ctx context.Context, userID, orderID string) (*Order, error)                                      // повторить заказ заново
}

// ReviewService - логика отзывов.
type ReviewService interface {
	Create(ctx context.Context, userID, venueID, orderID string, req CreateReviewRequest) (*Review, error) // оставить отзыв
	ListByVenue(ctx context.Context, venueID string, page, perPage int) ([]Review, int, error)             // отзывы заведения
}

// NotificationService - логика уведомлений.
type NotificationService interface {
	Create(ctx context.Context, notification *Notification) error                                  // создать уведомление
	ListByUser(ctx context.Context, userID string, page, perPage int) ([]Notification, int, error) // уведомления пользователя
	MarkAsRead(ctx context.Context, id string) error                                               // пометить прочитанным
	RegisterPushToken(ctx context.Context, token *PushToken) error                                 // сохранить пуш-токен
}
