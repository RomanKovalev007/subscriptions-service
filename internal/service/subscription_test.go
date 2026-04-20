package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/RomanKovalev007/subscriptions-service/internal/apperr"
	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepo — ручной мок репозитория.
type mockRepo struct {
	createFn    func(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error)
	getByIDFn   func(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	listFn      func(ctx context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error)
	updateFn    func(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error)
	deleteFn    func(ctx context.Context, id uuid.UUID) error
	totalCostFn func(ctx context.Context, f domain.TotalCostFilter) (int, error)
}

func (m *mockRepo) Create(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error) {
	return m.createFn(ctx, s)
}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockRepo) List(ctx context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error) {
	return m.listFn(ctx, userID, serviceName)
}
func (m *mockRepo) Update(ctx context.Context, s *domain.Subscription) (*domain.Subscription, error) {
	return m.updateFn(ctx, s)
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}
func (m *mockRepo) TotalCost(ctx context.Context, f domain.TotalCostFilter) (int, error) {
	return m.totalCostFn(ctx, f)
}

func newTestService(repo SubscriptionRepository) *subscriptionService {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewSubscriptionService(repo, logger)
}

func ptr[T any](v T) *T { return &v }

var (
	testUserID = uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba")
	testSubID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	errDB      = errors.New("database error")
)

// ─── Create ───────────────────────────────────────────────────────────────────

func TestCreate_Success_NoEndDate(t *testing.T) {
	want := &domain.Subscription{ID: testSubID, ServiceName: "Yandex Plus", Price: 400}
	svc := newTestService(&mockRepo{
		createFn: func(_ context.Context, s *domain.Subscription) (*domain.Subscription, error) {
			assert.Equal(t, "Yandex Plus", s.ServiceName)
			assert.Equal(t, 400, s.Price)
			assert.Nil(t, s.EndDate)
			return want, nil
		},
	})

	got, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      testUserID,
		StartDate:   "07-2025",
	})

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCreate_Success_WithEndDate(t *testing.T) {
	svc := newTestService(&mockRepo{
		createFn: func(_ context.Context, s *domain.Subscription) (*domain.Subscription, error) {
			require.NotNil(t, s.EndDate)
			assert.True(t, s.EndDate.After(s.StartDate))
			return s, nil
		},
	})

	_, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     ptr("09-2025"),
	})

	require.NoError(t, err)
}

func TestCreate_InvalidStartDate(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      testUserID,
		StartDate:   "2025-07", // неверный формат
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
}

func TestCreate_InvalidEndDate(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     ptr("invalid"),
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
}

func TestCreate_EndDateSameAsStartDate(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     ptr("07-2025"),
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
	assert.Contains(t, appErr.Message, "end_date must be after start_date")
}

func TestCreate_EndDateBeforeStartDate(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     ptr("06-2025"),
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
}

func TestCreate_RepoError(t *testing.T) {
	svc := newTestService(&mockRepo{
		createFn: func(_ context.Context, _ *domain.Subscription) (*domain.Subscription, error) {
			return nil, errDB
		},
	})

	_, err := svc.Create(context.Background(), domain.CreateSubscriptionInput{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      testUserID,
		StartDate:   "07-2025",
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInternalError, appErr.Code)
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func TestGetByID_Success(t *testing.T) {
	want := &domain.Subscription{ID: testSubID}
	svc := newTestService(&mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Subscription, error) {
			assert.Equal(t, testSubID, id)
			return want, nil
		},
	})

	got, err := svc.GetByID(context.Background(), testSubID)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newTestService(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
			return nil, apperr.ErrNotFound
		},
	})

	_, err := svc.GetByID(context.Background(), testSubID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeNotFound, appErr.Code)
}

func TestGetByID_RepoError(t *testing.T) {
	svc := newTestService(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
			return nil, errDB
		},
	})

	_, err := svc.GetByID(context.Background(), testSubID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInternalError, appErr.Code)
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestList_Success(t *testing.T) {
	want := []*domain.Subscription{{ID: testSubID}, {ID: uuid.New()}}
	svc := newTestService(&mockRepo{
		listFn: func(_ context.Context, userID uuid.UUID, serviceName *string) ([]*domain.Subscription, error) {
			assert.Equal(t, testUserID, userID)
			assert.Nil(t, serviceName)
			return want, nil
		},
	})

	got, err := svc.List(context.Background(), testUserID, nil)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestList_WithServiceNameFilter(t *testing.T) {
	svc := newTestService(&mockRepo{
		listFn: func(_ context.Context, _ uuid.UUID, serviceName *string) ([]*domain.Subscription, error) {
			require.NotNil(t, serviceName)
			assert.Equal(t, "Netflix", *serviceName)
			return []*domain.Subscription{}, nil
		},
	})

	_, err := svc.List(context.Background(), testUserID, ptr("Netflix"))
	require.NoError(t, err)
}

func TestList_Empty(t *testing.T) {
	svc := newTestService(&mockRepo{
		listFn: func(_ context.Context, _ uuid.UUID, _ *string) ([]*domain.Subscription, error) {
			return []*domain.Subscription{}, nil
		},
	})

	got, err := svc.List(context.Background(), testUserID, nil)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestList_RepoError(t *testing.T) {
	svc := newTestService(&mockRepo{
		listFn: func(_ context.Context, _ uuid.UUID, _ *string) ([]*domain.Subscription, error) {
			return nil, errDB
		},
	})

	_, err := svc.List(context.Background(), testUserID, nil)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInternalError, appErr.Code)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
	want := &domain.Subscription{ID: testSubID, ServiceName: "Yandex Plus", Price: 500}
	svc := newTestService(&mockRepo{
		updateFn: func(_ context.Context, s *domain.Subscription) (*domain.Subscription, error) {
			assert.Equal(t, testSubID, s.ID)
			return want, nil
		},
	})

	got, err := svc.Update(context.Background(), testSubID, domain.UpdateSubscriptionInput{
		ServiceName: "Yandex Plus",
		Price:       500,
		UserID:      testUserID,
		StartDate:   "07-2025",
	})

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUpdate_InvalidStartDate(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Update(context.Background(), testSubID, domain.UpdateSubscriptionInput{
		ServiceName: "Yandex Plus",
		Price:       500,
		UserID:      testUserID,
		StartDate:   "bad-date",
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
}

func TestUpdate_EndDateSameAsStartDate(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.Update(context.Background(), testSubID, domain.UpdateSubscriptionInput{
		ServiceName: "Yandex Plus",
		Price:       500,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     ptr("07-2025"),
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
}

func TestUpdate_NotFound(t *testing.T) {
	svc := newTestService(&mockRepo{
		updateFn: func(_ context.Context, _ *domain.Subscription) (*domain.Subscription, error) {
			return nil, apperr.ErrNotFound
		},
	})

	_, err := svc.Update(context.Background(), testSubID, domain.UpdateSubscriptionInput{
		ServiceName: "Yandex Plus",
		Price:       500,
		UserID:      testUserID,
		StartDate:   "07-2025",
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeNotFound, appErr.Code)
}

func TestUpdate_RepoError(t *testing.T) {
	svc := newTestService(&mockRepo{
		updateFn: func(_ context.Context, _ *domain.Subscription) (*domain.Subscription, error) {
			return nil, errDB
		},
	})

	_, err := svc.Update(context.Background(), testSubID, domain.UpdateSubscriptionInput{
		ServiceName: "Yandex Plus",
		Price:       500,
		UserID:      testUserID,
		StartDate:   "07-2025",
	})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInternalError, appErr.Code)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestDelete_Success(t *testing.T) {
	svc := newTestService(&mockRepo{
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, testSubID, id)
			return nil
		},
	})

	err := svc.Delete(context.Background(), testSubID)
	require.NoError(t, err)
}

func TestDelete_NotFound(t *testing.T) {
	svc := newTestService(&mockRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return apperr.ErrNotFound
		},
	})

	err := svc.Delete(context.Background(), testSubID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeNotFound, appErr.Code)
}

func TestDelete_RepoError(t *testing.T) {
	svc := newTestService(&mockRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return errDB
		},
	})

	err := svc.Delete(context.Background(), testSubID)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInternalError, appErr.Code)
}

// ─── TotalCost ────────────────────────────────────────────────────────────────

func TestTotalCost_Success(t *testing.T) {
	svc := newTestService(&mockRepo{
		totalCostFn: func(_ context.Context, f domain.TotalCostFilter) (int, error) {
			assert.Equal(t, testUserID, f.UserID)
			assert.Nil(t, f.ServiceName)
			return 1200, nil
		},
	})

	total, err := svc.TotalCost(context.Background(), "01-2025", "03-2025", testUserID, nil)

	require.NoError(t, err)
	assert.Equal(t, 1200, total)
}

func TestTotalCost_WithServiceNameFilter(t *testing.T) {
	svc := newTestService(&mockRepo{
		totalCostFn: func(_ context.Context, f domain.TotalCostFilter) (int, error) {
			require.NotNil(t, f.ServiceName)
			assert.Equal(t, "Netflix", *f.ServiceName)
			return 800, nil
		},
	})

	total, err := svc.TotalCost(context.Background(), "01-2025", "03-2025", testUserID, ptr("Netflix"))

	require.NoError(t, err)
	assert.Equal(t, 800, total)
}

func TestTotalCost_SameMonth(t *testing.T) {
	// from == to — один месяц, допустимо
	svc := newTestService(&mockRepo{
		totalCostFn: func(_ context.Context, f domain.TotalCostFilter) (int, error) {
			assert.Equal(t, f.From, f.To)
			return 400, nil
		},
	})

	total, err := svc.TotalCost(context.Background(), "07-2025", "07-2025", testUserID, nil)

	require.NoError(t, err)
	assert.Equal(t, 400, total)
}

func TestTotalCost_CorrectDateRange(t *testing.T) {
	svc := newTestService(&mockRepo{
		totalCostFn: func(_ context.Context, f domain.TotalCostFilter) (int, error) {
			wantFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			wantTo := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
			assert.Equal(t, wantFrom, f.From)
			assert.Equal(t, wantTo, f.To)
			return 0, nil
		},
	})

	_, err := svc.TotalCost(context.Background(), "01-2025", "03-2025", testUserID, nil)
	require.NoError(t, err)
}

func TestTotalCost_InvalidFromFormat(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.TotalCost(context.Background(), "2025-01", "03-2025", testUserID, nil)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
	assert.Contains(t, appErr.Message, "from")
}

func TestTotalCost_InvalidToFormat(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.TotalCost(context.Background(), "01-2025", "bad", testUserID, nil)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
	assert.Contains(t, appErr.Message, "to")
}

func TestTotalCost_ToBeforeFrom(t *testing.T) {
	svc := newTestService(&mockRepo{})

	_, err := svc.TotalCost(context.Background(), "06-2025", "01-2025", testUserID, nil)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
	assert.Contains(t, appErr.Message, "to must be after from")
}

func TestTotalCost_RepoError(t *testing.T) {
	svc := newTestService(&mockRepo{
		totalCostFn: func(_ context.Context, _ domain.TotalCostFilter) (int, error) {
			return 0, errDB
		},
	})

	_, err := svc.TotalCost(context.Background(), "01-2025", "03-2025", testUserID, nil)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperr.CodeInternalError, appErr.Code)
}
