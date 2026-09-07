package domain

// Лимиты пагинации по умолчанию.
const (
	DefaultPage    = 1   // первая страница
	DefaultPerPage = 20  // записей на странице
	MaxPerPage     = 100 // больше за раз не отдаём
)

// NormalizePagination - приводит page/perPage к допустимым значениям.
func NormalizePagination(page, perPage int) (int, int) {
	if page <= 0 {
		page = DefaultPage
	}
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

// TotalPages - считает общее число страниц по total и perPage.
func TotalPages(total, perPage int) int {
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	pages := total / perPage
	if total%perPage > 0 {
		pages++
	}
	return pages
}
