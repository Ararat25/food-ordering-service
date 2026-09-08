# Авито.Кухня — Backend Trainee Assignment

MVP-сервис доставки еды. Пользователи могут просматривать заведения, делать заказы, отслеживать статус и оставлять отзывы. Заведения управляют меню и заказами через отдельный API.

## Архитектура

### Обзор (C4 Level 2)

```
┌─────────────────────────────────────────────────────────┐
│                    Клиент (Web)                         │
└───────────────────────────┬─────────────────────────────┘
                            │ HTTP
┌───────────────────────────▼─────────────────────────────┐
│                  Основной сервис (app)                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │
│  │  Router  │ │ Handler  │ │ Service  │ │Repository│    │
│  │  (chi)   │ │          │ │          │ │(squirrel)│    │
│  └──────────┘ └──────────┘ └──────────┘ └─────┬────┘    │
└───────────────────────────────────────────────┼─────────┘
                                                │
                                    ┌───────────▼─────────┐
                                    │    PostgreSQL       │
                                    └─────────────────────┘

┌─────────────────────────────────────────────────────────┐
│              Venue-сервис (пример заведения)            │
│  Опрашивает основной API (GET /api/v1/venue/orders)     │
│  и принимает pending-заказы (PATCH .../accept)          │
└─────────────────────────────────────────────────────────┘
```

### Структура проекта

```
/
├── cmd/
│   ├── server/main.go         — точка входа основного сервиса
│   └── venue/main.go          — точка входа venue-сервиса
├── internal/
│   ├── api/
│   │   ├── handler/           — HTTP-обработчики
│   │   ├── middleware/         — RequestID, Logging, CORS, Auth
│   │   └── router/            — настройка маршрутов
│   ├── config/                — загрузка конфигурации из env
│   ├── db/                    — подключение к PostgreSQL
│   ├── domain/                — бизнес-сущности и интерфейсы
│   ├── repository/            — доступ к БД
│   └── service/               — бизнес-логика
├── migrations/                — SQL-миграции
├── docs/                      — CJM, OpenAPI, архитектура
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── .env.example
```

## Стек

| Компонент | Технология |
|-----------|------------|
| Язык | Go 1.26 |
| HTTP-роутер | go-chi/chi |
| БД | PostgreSQL 16 |
| SQL query builder | squirrel |
| Миграции | goose |
| Контейнеризация | Docker, Docker Compose |
| Линтер | golangci-lint |

## Быстрый старт

```bash
# 1. Скопировать .env
cp .env.example .env

# 2. Запустить через Docker Compose
make docker-up

# 3. Проверить работоспособность
curl http://localhost:8080/health
```

## Доступные команды

```bash
make build        # Собрать бинарники
make run          # Запустить основной сервис
make run-venue    # Запустить venue-сервис
make test         # Запустить тесты
make lint         # Запустить линтер
make docker-up    # Запустить через Docker
make docker-down  # Остановить Docker
```

## API

Списки отдаются постранично: `?page=&per_page=`, ответ — `{data, pagination}`.
Все денежные суммы (`price`, `total_amount`, `amount`) — целые копейки (`int64`).

### Пользовательское API

| Метод | Описание |
|-------|----------|
| `GET /api/v1/venues` | Список заведений |
| `GET /api/v1/venues/:id` | Детали заведения |
| `GET /api/v1/venues/:id/menu` | Меню заведения |
| `GET /api/v1/venues/:id/reviews` | Отзывы заведения |
| `POST /api/v1/venues/:id/reviews` | Оставить отзыв |
| `GET /api/v1/cart?venue_id=` | Корзина |
| `POST /api/v1/cart/items?venue_id=` | Добавить в корзину |
| `PUT /api/v1/cart/items/:id` | Изменить количество |
| `DELETE /api/v1/cart/items/:id` | Убрать из корзины |
| `DELETE /api/v1/cart?venue_id=` | Очистить корзину |
| `POST /api/v1/orders` | Создать заказ |
| `GET /api/v1/orders` | Список заказов |
| `GET /api/v1/orders/:id` | Детали заказа |
| `GET /api/v1/orders/:id/track` | Отслеживание |
| `POST /api/v1/orders/:id/repeat` | Повторить заказ |
| `POST /api/v1/notifications/register` | Push-токен |
| `GET /api/v1/notifications` | Уведомления |

### API заведения (требует API-ключ)

| Метод | Описание |
|-------|----------|
| `GET /api/v1/venue/menu` | Меню заведения |
| `POST /api/v1/venue/menu` | Добавить блюдо |
| `PUT /api/v1/venue/menu/:id` | Обновить блюдо |
| `DELETE /api/v1/venue/menu/:id` | Удалить блюдо |
| `PATCH /api/v1/venue/menu/:id/availability` | Наличие блюда |
| `GET /api/v1/venue/orders` | Заказы заведения |
| `GET /api/v1/venue/orders/:id` | Детали заказа |
| `PATCH /api/v1/venue/orders/:id/accept` | Принять заказ |
| `PATCH /api/v1/venue/orders/:id/reject` | Отклонить заказ |
| `PATCH /api/v1/venue/orders/:id/status` | Сменить статус |

### Аутентификация

- **Пользователи:** заголовок `X-User-ID` (заглушка)
- **Заведения:** `Authorization: Bearer <api_key>`

## Схема БД

### Основные таблицы

- `users` — пользователи
- `venues` — заведения
- `menu_categories` — категории меню
- `menu_items` — блюда
- `cart_items` — корзина
- `orders` — заказы
- `order_items` — позиции заказа
- `payments` — платежи
- `reviews` — отзывы
- `notifications` — уведомления

## Упрощения для MVP

Осознанные упрощения:

- Авторизация пользователей — заглушка (X-User-ID)
- Оплата — имитация без реальной интеграции
- Push-уведомления — заглушка (токены только регистрируются)
- Нет системы промокодов/скидок
- Нет аналитики для заведений
- Seed-данные для демонстрации

## License

MIT
