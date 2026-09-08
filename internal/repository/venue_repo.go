package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"avito-kitchen/internal/domain"
)

// VenueRepo - заведения в базе данных.
type VenueRepo struct {
	db *sql.DB // соединение с базой
}

// NewVenueRepo - создаёт репозиторий заведений.
func NewVenueRepo(db *sql.DB) *VenueRepo {
	return &VenueRepo{db: db}
}

// Create - сохраняет новое заведение, дубль даёт ErrAlreadyExists.
func (r *VenueRepo) Create(ctx context.Context, venue *domain.Venue) error {
	query, args, err := sb.Insert("venues").
		Columns("id", "name", "description", "logo_url", "phone", "address",
			"latitude", "longitude", "working_hours", "delivery_zone", "is_active", "api_key").
		Values(venue.ID, venue.Name, venue.Description, venue.LogoURL, venue.Phone, venue.Address,
			venue.Latitude, venue.Longitude, venue.WorkingHours, venue.DeliveryZone, venue.IsActive, venue.APIKey).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert venue: %w", err)
	}

	return nil
}

// GetByID - находит заведение по id.
func (r *VenueRepo) GetByID(ctx context.Context, id string) (*domain.Venue, error) {
	query, args, err := sb.Select(venueColumns...).From("venues").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var venue domain.Venue
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&venue.ID, &venue.Name, &venue.Description, &venue.LogoURL, &venue.Phone,
		&venue.Address, &venue.Latitude, &venue.Longitude, &venue.WorkingHours,
		&venue.DeliveryZone, &venue.IsActive, &venue.APIKey, &venue.CreatedAt, &venue.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get venue by id: %w", err)
	}

	return &venue, nil
}

// GetByAPIKey - находит заведение по API-ключу, нужно для auth заведений.
func (r *VenueRepo) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Venue, error) {
	query, args, err := sb.Select(venueColumns...).From("venues").Where(sq.Eq{"api_key": apiKey}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var venue domain.Venue
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&venue.ID, &venue.Name, &venue.Description, &venue.LogoURL, &venue.Phone,
		&venue.Address, &venue.Latitude, &venue.Longitude, &venue.WorkingHours,
		&venue.DeliveryZone, &venue.IsActive, &venue.APIKey, &venue.CreatedAt, &venue.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get venue by api key: %w", err)
	}

	return &venue, nil
}

// List - возвращает активные заведения с поиском и пагинацией плюс общее число.
func (r *VenueRepo) List(ctx context.Context, filters domain.VenueFilters) ([]domain.Venue, int, error) {
	baseQuery := sb.Select(venueColumns...).From("venues").Where(sq.Eq{"is_active": true})
	countQuery := sb.Select("COUNT(*)").From("venues").Where(sq.Eq{"is_active": true})

	if filters.Search != "" {
		search := "%" + strings.ToLower(filters.Search) + "%"
		condition := sq.Or{
			sq.Like{"LOWER(name)": search},
			sq.Like{"LOWER(description)": search},
		}
		baseQuery = baseQuery.Where(condition)
		countQuery = countQuery.Where(condition)
	}

	var total int
	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count venues: %w", err)
	}

	offset := (filters.Page - 1) * filters.PerPage
	baseQuery = baseQuery.OrderBy("name ASC").Limit(uint64(filters.PerPage)).Offset(uint64(offset))

	query, args, err := baseQuery.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list venues: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var venues []domain.Venue
	for rows.Next() {
		var v domain.Venue
		if err := rows.Scan(
			&v.ID, &v.Name, &v.Description, &v.LogoURL, &v.Phone,
			&v.Address, &v.Latitude, &v.Longitude, &v.WorkingHours,
			&v.DeliveryZone, &v.IsActive, &v.APIKey, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan venue: %w", err)
		}
		venues = append(venues, v)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate venues: %w", err)
	}

	return venues, total, nil
}

// Update - обновляет заведение, чужой id даёт ErrNotFound.
func (r *VenueRepo) Update(ctx context.Context, venue *domain.Venue) error {
	query, args, err := sb.Update("venues").
		Set("name", venue.Name).
		Set("description", venue.Description).
		Set("logo_url", venue.LogoURL).
		Set("phone", venue.Phone).
		Set("address", venue.Address).
		Set("latitude", venue.Latitude).
		Set("longitude", venue.Longitude).
		Set("working_hours", venue.WorkingHours).
		Set("delivery_zone", venue.DeliveryZone).
		Set("is_active", venue.IsActive).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": venue.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update venue: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
