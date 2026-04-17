package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type subscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *subscriptionRepo {
	return &subscriptionRepo{pool: pool}
}

func (r *subscriptionRepo) Create(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error) {
	const q = `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate)

	return scanSubscription(row)
}

func (r *subscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	const q = `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	return scanSubscription(row)
}

func (r *subscriptionRepo) List(ctx context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error) {
	const q = `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1
		  AND ($2 IS NULL OR service_name = $2)
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, userID, serviceName)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list subscriptions rows: %w", err)
	}
	return subs, nil
}

func (r *subscriptionRepo) Update(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error) {
	const q = `
		UPDATE subscriptions
		SET service_name = $1,
		    price        = $2,
		    user_id      = $3,
		    start_date   = $4,
		    end_date     = $5,
		    updated_at   = NOW()
		WHERE id = $6
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate, s.ID)
	return scanSubscription(row)
}

func (r *subscriptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM subscriptions WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type subscriptionScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(scanner subscriptionScanner) (*domain.Subscription, error) {
	var s domain.Subscription
	if err := scanner.Scan(
		&s.ID,
		&s.ServiceName,
		&s.Price,
		&s.UserID,
		&s.StartDate,
		&s.EndDate,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}
