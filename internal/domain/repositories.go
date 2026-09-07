package domain

import "context"

// UserRepository - работа с пользователями в БД.
type UserRepository interface {
	Create(ctx context.Context, user *User) error                // сохранить нового
	GetByID(ctx context.Context, id string) (*User, error)       // найти по id
	GetByEmail(ctx context.Context, email string) (*User, error) // найти по email
	Update(ctx context.Context, user *User) error                // обновить
}

// VenueRepository - работа с заведениями в БД.
type VenueRepository interface {
	Create(ctx context.Context, venue *Venue) error                       // сохранить новое
	GetByID(ctx context.Context, id string) (*Venue, error)               // найти по id
	GetByAPIKey(ctx context.Context, apiKey string) (*Venue, error)       // найти по API-ключу
	List(ctx context.Context, filters VenueFilters) ([]Venue, int, error) // список с фильтрами, возвращает записи и общее число
	Update(ctx context.Context, venue *Venue) error                       // обновить
}

// VenueFilters - фильтры и пагинация для списка заведений.
type VenueFilters struct {
	Search   string // поиск по названию и описанию
	IsActive *bool  // только активные, nil — все
	Page     int    // страница
	PerPage  int    // записей на странице
}

// MenuCategoryRepository - работа с категориями меню в БД.
type MenuCategoryRepository interface {
	Create(ctx context.Context, category *MenuCategory) error                // сохранить новую
	GetByID(ctx context.Context, id string) (*MenuCategory, error)           // найти по id
	ListByVenue(ctx context.Context, venueID string) ([]MenuCategory, error) // все категории заведения
	Update(ctx context.Context, category *MenuCategory) error                // обновить
	Delete(ctx context.Context, id string) error                             // удалить
}

// MenuItemRepository - работа с блюдами в БД.
type MenuItemRepository interface {
	Create(ctx context.Context, item *MenuItem) error                          // сохранить новое
	GetByID(ctx context.Context, id string) (*MenuItem, error)                 // найти по id
	ListByVenue(ctx context.Context, venueID string) ([]MenuItem, error)       // все блюда заведения
	ListByCategory(ctx context.Context, categoryID string) ([]MenuItem, error) // блюда одной категории
	Update(ctx context.Context, item *MenuItem) error                          // обновить
	Delete(ctx context.Context, id string) error                               // удалить
	UpdateAvailability(ctx context.Context, id string, available bool) error   // сменить наличие
}

// CartRepository - работа с корзинами в БД.
type CartRepository interface {
	GetOrCreate(ctx context.Context, userID, venueID string) (*Cart, error)    // найти корзину или создать
	GetByID(ctx context.Context, id string) (*Cart, error)                     // найти по id
	Clear(ctx context.Context, cartID string) error                            // убрать все позиции
	AddItem(ctx context.Context, item *CartItem) error                         // добавить позицию
	UpdateItemQuantity(ctx context.Context, itemID string, quantity int) error // сменить количество
	RemoveItem(ctx context.Context, itemID string) error                       // убрать позицию
	GetItems(ctx context.Context, cartID string) ([]CartItem, error)           // все позиции корзины
}

// OrderRepository - работа с заказами в БД.
type OrderRepository interface {
	Create(ctx context.Context, order *Order) error                                                          // сохранить заказ
	CreateWithItems(ctx context.Context, order *Order, items []OrderItem) error                              // сохранить заказ с позициями и платежом в одной транзакции
	GetByID(ctx context.Context, id string) (*Order, error)                                                  // найти по id
	ListByUser(ctx context.Context, userID string, page, perPage int) ([]Order, int, error)                  // заказы пользователя
	ListByVenue(ctx context.Context, venueID string, status string, page, perPage int) ([]Order, int, error) // заказы заведения
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error                                   // сменить статус
	GetItems(ctx context.Context, orderID string) ([]OrderItem, error)                                       // позиции заказа
}

// PaymentRepository - работа с оплатами в БД.
type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error                      // сохранить новую
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)      // найти по заказу
	UpdateStatus(ctx context.Context, id string, status PaymentStatus) error // сменить статус
}

// ReviewRepository - работа с отзывами в БД.
type ReviewRepository interface {
	Create(ctx context.Context, review *Review) error                                          // сохранить новый
	ListByVenue(ctx context.Context, venueID string, page, perPage int) ([]Review, int, error) // отзывы заведения
	GetByUserAndOrder(ctx context.Context, userID, orderID string) (*Review, error)            // отзыв пользователя по заказу
}

// NotificationRepository - работа с уведомлениями в БД.
type NotificationRepository interface {
	Create(ctx context.Context, notification *Notification) error                                  // сохранить новое
	ListByUser(ctx context.Context, userID string, page, perPage int) ([]Notification, int, error) // уведомления пользователя
	MarkAsRead(ctx context.Context, id string) error                                               // пометить прочитанным
}

// PushTokenRepository - работа с пуш-токенами в БД.
type PushTokenRepository interface {
	Register(ctx context.Context, token *PushToken) error                // сохранить токен
	GetByUserID(ctx context.Context, userID string) ([]PushToken, error) // все токены пользователя
	Delete(ctx context.Context, token string) error                      // удалить токен
}
