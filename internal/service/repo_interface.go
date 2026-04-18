package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	List(ctx context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error)
	Update(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	TotalCost(ctx context.Context, f domain.TotalCostFilter) (int, error)
}