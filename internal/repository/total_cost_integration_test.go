package repository_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RomanKovalev007/subscriptions-service/internal/domain"
	"github.com/RomanKovalev007/subscriptions-service/internal/repository"
	"github.com/RomanKovalev007/subscriptions-service/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("get dsn: %v", err)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("migrations source: %v", err)
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	if err := migrator.Up(); err != nil {
		log.Fatalf("migrate up: %v", err)
	}
	migrator.Close()

	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("create pool: %v", err)
	}
	defer testPool.Close()

	os.Exit(m.Run())
}

// clearSubscriptions очищает таблицу перед каждым тестом.
func clearSubscriptions(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "DELETE FROM subscriptions")
	require.NoError(t, err)
}

func insertSub(t *testing.T, userID uuid.UUID, serviceName string, price int, start, end string) {
	t.Helper()
	startDate, err := time.Parse("01-2006", start)
	require.NoError(t, err)

	var endDate *time.Time
	if end != "" {
		d, err := time.Parse("01-2006", end)
		require.NoError(t, err)
		endDate = &d
	}

	_, err = testPool.Exec(context.Background(),
		`INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5)`,
		serviceName, price, userID, startDate, endDate,
	)
	require.NoError(t, err)
}

func makeFilter(userID uuid.UUID, from, to string, serviceName *string) domain.TotalCostFilter {
	fromDate, _ := time.Parse("01-2006", from)
	toDate, _ := time.Parse("01-2006", to)
	return domain.TotalCostFilter{From: fromDate, To: toDate, UserID: userID, ServiceName: serviceName}
}

func strPtr(s string) *string { return &s }

// ─── Тесты ───────────────────────────────────────────────────────────────────

func TestIntegration_TotalCost_NoSubscriptions(t *testing.T) {
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)

	total, err := repo.TotalCost(context.Background(), makeFilter(uuid.New(), "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestIntegration_TotalCost_FullyInsidePeriod(t *testing.T) {
	// Подписка 02-2025..03-2025, период 01-2025..06-2025 → 2 * 400 = 800
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "02-2025", "03-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "06-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 800, total)
}

func TestIntegration_TotalCost_StartsBeforePeriod(t *testing.T) {
	// Подписка 01-2024..03-2025, период 01-2025..03-2025 → 3 * 400 = 1200
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "01-2024", "03-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 1200, total)
}

func TestIntegration_TotalCost_EndsAfterPeriod(t *testing.T) {
	// Подписка 01-2025..12-2025, период 01-2025..03-2025 → 3 * 500 = 1500
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Yandex Plus", 500, "01-2025", "12-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 1500, total)
}

func TestIntegration_TotalCost_OpenEndedSubscription(t *testing.T) {
	// Подписка 01-2025..NULL, период 01-2025..03-2025 → 3 * 300 = 900
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Spotify", 300, "01-2025", "")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 900, total)
}

func TestIntegration_TotalCost_SingleMonthPeriod(t *testing.T) {
	// from == to — один месяц: 1 * 200 = 200
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Spotify", 200, "03-2025", "")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "03-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 200, total)
}

func TestIntegration_TotalCost_MultipleSubscriptions(t *testing.T) {
	// 3*400 + 3*200 + 2*500 = 1200 + 600 + 1000 = 2800
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "01-2025", "03-2025")
	insertSub(t, userID, "Spotify", 200, "01-2025", "03-2025")
	insertSub(t, userID, "Yandex Plus", 500, "02-2025", "03-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 2800, total)
}

func TestIntegration_TotalCost_FilterByServiceName(t *testing.T) {
	// Только Netflix: 3 * 400 = 1200
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "01-2025", "03-2025")
	insertSub(t, userID, "Spotify", 200, "01-2025", "03-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", strPtr("Netflix")))

	require.NoError(t, err)
	assert.Equal(t, 1200, total)
}

func TestIntegration_TotalCost_SubscriptionEndedBeforePeriod(t *testing.T) {
	// Подписка закончилась до начала периода → 0
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "01-2024", "06-2024")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestIntegration_TotalCost_SubscriptionStartsAfterPeriod(t *testing.T) {
	// Подписка начинается после конца периода → 0
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "06-2025", "12-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestIntegration_TotalCost_AcrossYears(t *testing.T) {
	// Подписка 11-2024..02-2025, период 11-2024..02-2025 → 4 * 600 = 2400
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()

	insertSub(t, userID, "Netflix", 600, "11-2024", "02-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "11-2024", "02-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 2400, total)
}

func TestIntegration_TotalCost_UserIsolation(t *testing.T) {
	// Подписки другого пользователя не учитываются: только 3 * 400 = 1200
	clearSubscriptions(t)
	repo := repository.NewSubscriptionRepository(testPool)
	userID := uuid.New()
	otherUserID := uuid.New()

	insertSub(t, userID, "Netflix", 400, "01-2025", "03-2025")
	insertSub(t, otherUserID, "Netflix", 999, "01-2025", "03-2025")

	total, err := repo.TotalCost(context.Background(), makeFilter(userID, "01-2025", "03-2025", nil))

	require.NoError(t, err)
	assert.Equal(t, 1200, total)
}
