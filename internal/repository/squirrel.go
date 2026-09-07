package repository

import sq "github.com/Masterminds/squirrel"

// sb - сборщик SQL с плейсхолдерами вида $1, такие требует PostgreSQL.
// Обычные билдеры squirrel ставят "?", Postgres их не понимает.
var sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
