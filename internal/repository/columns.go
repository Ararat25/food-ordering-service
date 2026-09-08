package repository

// Явные списки колонок для SELECT-запросов. SELECT * ломается при любом
// изменении схемы (удалили колонку — упал Scan), явный список переживает
// лишние колонки и сразу показывает, что именно читаем.
// Порядок в каждом списке повторяет порядок Scan в репозиториях.
var (
	venueColumns = []string{
		"id", "name", "description", "logo_url", "phone", "address",
		"latitude", "longitude", "working_hours", "delivery_zone",
		"is_active", "api_key", "created_at", "updated_at",
	}

	menuCategoryColumns = []string{
		"id", "venue_id", "name", "sort_order", "created_at",
	}

	menuItemColumns = []string{
		"id", "category_id", "venue_id", "name", "description", "price",
		"image_url", "is_available", "options", "created_at", "updated_at",
	}

	cartColumns = []string{
		"id", "user_id", "venue_id", "created_at", "updated_at",
	}

	cartItemColumns = []string{
		"id", "cart_id", "menu_item_id", "quantity", "options", "created_at",
	}

	orderColumns = []string{
		"id", "user_id", "venue_id", "status", "delivery_type",
		"delivery_address", "delivery_time", "estimated_delivery_time",
		"comment", "total_amount", "created_at", "updated_at", "idempotency_key",
	}

	orderItemColumns = []string{
		"id", "order_id", "menu_item_id", "quantity", "price", "options", "created_at",
	}

	paymentColumns = []string{
		"id", "order_id", "status", "amount", "method", "created_at",
	}

	reviewColumns = []string{
		"id", "user_id", "venue_id", "order_id", "rating", "comment", "created_at",
	}

	// "type" - зарезервированное слово в PostgreSQL, без кавычек запрос не соберётся.
	notificationColumns = []string{
		"id", "user_id", `"type"`, "title", "body", "data", "is_read", "created_at",
	}

	pushTokenColumns = []string{
		"id", "user_id", "token", "platform", "created_at",
	}
)
