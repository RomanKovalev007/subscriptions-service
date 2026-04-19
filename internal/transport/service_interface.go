package transport

import (
	"context"

	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
	"github.com/google/uuid"
)

type SubscriptionService interface {
	Create(ctx context.Context, input domain.CreateSubscriptionInput) (*domain.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	List(ctx context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error)
	Update(ctx context.Context, id uuid.UUID, input domain.UpdateSubscriptionInput) (*domain.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	TotalCost(ctx context.Context, from, to string, userID uuid.UUID, serviceName *string) (int, error)
}