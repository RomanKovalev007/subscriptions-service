package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/RomanKovalev007/subscriptions-service/internal/apperr"
	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
	"github.com/google/uuid"
)

type subscriptionService struct {
	repo   SubscriptionRepository
	logger *slog.Logger
}

func NewSubscriptionService(repo SubscriptionRepository, logger *slog.Logger) *subscriptionService {
	return &subscriptionService{repo: repo, logger: logger}
}


func (s *subscriptionService) Create(ctx context.Context, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidInput, "invalid user_id")
	}

	startDate, err := firstOfMonth(input.StartDate)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidInput, err.Error())
	}

	sub := &domain.Subscription{
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      userID,
		StartDate:   startDate,
	}

	if input.EndDate != nil {
		endDate, err := firstOfMonth(*input.EndDate)
		if err != nil {
			return nil, apperr.New(apperr.CodeInvalidInput, err.Error())
		}
		if !endDate.After(startDate) {
			return nil, apperr.New(apperr.CodeInvalidInput, "end_date must be after start_date")
		}
		sub.EndDate = &endDate
	}

	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		s.logger.Error("create subscription", "error", err)
		return nil, apperr.New(apperr.CodeInternalError, "cant insert subscription")
	}

	return created, nil
}

func (s *subscriptionService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.New(apperr.CodeNotFound, "subscription info not found")
		}
		s.logger.Error("get subscription", "id", id.String(),"error", err)
		return nil, apperr.New(apperr.CodeInternalError, "cant get subscription info")
	}
	return sub, nil
}

func (s *subscriptionService) List(ctx context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error) {
	subs, err := s.repo.List(ctx, userID, serviceName)
	if err != nil {
		s.logger.Error("list subscriptions", "error", err)
		return nil, apperr.New(apperr.CodeInternalError, "cant get list of subscription info")
	}
	return subs, nil
}

func (s *subscriptionService) Update(ctx context.Context, id uuid.UUID, input domain.UpdateSubscriptionInput) (*domain.Subscription, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidInput, "invalid user_id")
	}

	startDate, err := firstOfMonth(input.StartDate)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidInput, err.Error())
	}

	sub := &domain.Subscription{
		ID:          id,
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      userID,
		StartDate:   startDate,
	}

	if input.EndDate != nil {
		endDate, err := firstOfMonth(*input.EndDate)
		if err != nil {
			return nil, apperr.New(apperr.CodeInvalidInput, err.Error())
		}
		if !endDate.After(startDate) {
			return nil, apperr.New(apperr.CodeInvalidInput, "end_date must be after start_date")
		}
		sub.EndDate = &endDate
	}

	updated, err := s.repo.Update(ctx, sub)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.New(apperr.CodeNotFound, "subscription info not found")
		}
		s.logger.Error("update subscription", "id", id.String(),"error", err)
		return nil, apperr.New(apperr.CodeInternalError, "cant update subscription info")
	}
	return updated, nil
}

func (s *subscriptionService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.New(apperr.CodeNotFound, "subscription info not found")
		}
		s.logger.Error("delete subscription", "id", id.String(),"error", err)
		return apperr.New(apperr.CodeInternalError, "cant delete subscription info")
	}
	return nil
}

func (s *subscriptionService) TotalCost(ctx context.Context, input domain.TotalCostInput) (int, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidInput, "invalid user_id")
	}
	fromDate, err := firstOfMonth(input.From)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidInput, "from: "+err.Error())
	}
	toDate, err := firstOfMonth(input.To)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidInput, "to: "+err.Error())
	}
	if toDate.Before(fromDate) {
		return 0, apperr.New(apperr.CodeInvalidInput, "to must be after from")
	}

	filter := domain.TotalCostFilter{
		From:        fromDate,
		To:          toDate,
		UserID:      userID,
		ServiceName: input.ServiceName,
	}

	total, err := s.repo.TotalCost(ctx, filter)
	if err != nil {
		s.logger.Error("total cost", "error", err)
		return 0, apperr.New(apperr.CodeInternalError, "cant calculate total cost")
	}
	return total, nil
}

// firstOfMonth parses "MM-YYYY" into the first day of that month.
func firstOfMonth(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q, expected MM-YYYY: %w", s, err)
	}
	return t, nil
}