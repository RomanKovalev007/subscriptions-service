package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          uuid.UUID  `db:"id"           json:"id"`
	ServiceName string     `db:"service_name" json:"service_name"`
	Price       int        `db:"price"        json:"price"`
	UserID      uuid.UUID  `db:"user_id"      json:"user_id"`
	StartDate   time.Time  `db:"start_date"   json:"start_date"`
	EndDate     *time.Time `db:"end_date"     json:"end_date,omitempty"`
	CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"   json:"updated_at"`
}

// CreateSubscriptionInput is the request body for creating a subscription.
type CreateSubscriptionInput struct {
	ServiceName string  `json:"service_name" validate:"required"`
	Price       int     `json:"price"        validate:"min=0"`
	UserID      string  `json:"user_id"      validate:"required,uuid4"`
	StartDate   string  `json:"start_date"   validate:"required"`
	EndDate     *string `json:"end_date"`
}

// UpdateSubscriptionInput is the request body for updating a subscription.
type UpdateSubscriptionInput struct {
	ServiceName string  `json:"service_name" validate:"required"`
	Price       int     `json:"price"        validate:"min=0"`
	UserID      string  `json:"user_id"      validate:"required,uuid4"`
	StartDate   string  `json:"start_date"   validate:"required"`
	EndDate     *string `json:"end_date"`
}

// TotalCostInput is the request body for the total-cost query.
type TotalCostInput struct {
	From        string  `json:"from" validate:"required"`
	To          int     `json:"to"`
	UserID      string  `json:"user_id" validate:"required,uuid4"`
	ServiceName *string `json:"service_name" `
}

// TotalCostFilter holds parameters for the total-cost query.
type TotalCostFilter struct {
	From        time.Time
	To          time.Time
	UserID      uuid.UUID
	ServiceName *string
}
