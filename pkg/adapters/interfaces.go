package adapters

import (
	"context"
	"github.com/shopspring/decimal"
	"niltonkummer/rinha-2025/pkg/models"
	"time"
)

type StatsAdapter interface {
	InsertStats(correlationID string, amount decimal.Decimal, fallback bool, requestedAt time.Time) error    // Method to insert stats into the database
	RetrieveTotalStatsByPeriod(fallback bool, start, end time.Time) (*models.PaymentsSummaryResponse, error) // Method to retrieve stats by period
}

type QueueAdapter interface {
	Enqueue(ctx context.Context, message any) error // Method to enqueue a message into the queue
	Dequeue(ctx context.Context) (any, error)       // Method to dequeue a message from the queue
}

type Cache interface {
	Add(value any)
	GetList() ([]any, error)
}
