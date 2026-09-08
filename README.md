# Авито.Кухня

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
| Язык | Go 1.26.9 |
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
Ошибки — `{"error", "code", "request_id"}`; 500-е скрывают детали, полные тексты в логах.
Создание заказа идемпотентно по паре `(X-User-ID, Idempotency-Key)`: ключ необязательный, если задан — до 255 printable ASCII bytes; повтор возвращает готовый заказ.
Тела запросов ограничены 1 МБ: лишние JSON-значения и неизвестные поля получают 400, превышение размера — 413.
Неверные `page`, `per_page` и `available` отклоняются с 400. В Compose внутренние порты API/venue фиксированы на 8080/8081; опубликованные host-порты задаются `APP_HOST_PORT` и `VENUE_HOST_PORT`. При локальном запуске Go-приложения загружают необязательный `.env` через `godotenv` перед чтением конфигурации; уже заданные переменные окружения имеют приоритет. Для CLI Goose Makefile также экспортирует `.env`.
Опции блюда передаются объектом выбранных значений (например, `{"size":"large"}`), сверяются с опциями блюда и сохраняются в корзине/заказе; в текущем MVP они не меняют базовую цену.

### Пользовательское API

| Метод | Описание |
|-------|----------|
| `GET /api/v1/venues` | Список заведений |
| `GET /api/v1/venues/:id` | Детали заведения |
| `GET /api/v1/venues/:id/menu` | Меню заведения (search, available, пагинация) |
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
| `DELETE /api/v1/notifications/register` | Отозвать свой push-токен |
| `GET /api/v1/notifications` | Уведомления |

### API заведения (требует API-ключ)

| Метод | Описание |
|-------|----------|
| `GET /api/v1/venue/menu` | Меню заведения (search, available, пагинация) |
| `GET /api/v1/venue/menu/categories` | Категории меню |
| `POST /api/v1/venue/menu/categories` | Создать категорию (`{"name":"Пицца","sort_order":1}`) |
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

- **Пользователи:** заголовок `X-User-ID` (заглушка, не аутентификация). Для локального демо после миграций доступны seed IDs `550e8400-e29b-41d4-a716-446655440001` и `550e8400-e29b-41d4-a716-446655440002`; иные ID нужно предварительно создать в `users`.
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
