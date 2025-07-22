package cache

import (
	"context"
	"github.com/shopspring/decimal"
	"niltonkummer/rinha-2025/pkg/models"
	"testing"
)

func TestQueue_EnqueueDequeue(t *testing.T) {
	ctx := context.Background()
	q := NewQueue(0)

	payment := models.PaymentRequest{Amount: decimal.New(100, 0)}
	if err := q.Enqueue(ctx, payment); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	got, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if got.Amount != payment.Amount {
		t.Errorf("Expected %v, got %v", payment.Amount, got.Amount)
	}
}

func TestQueue_DequeueEmpty(t *testing.T) {
	ctx := context.Background()
	q := NewQueue(0)

	got, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue on empty queue failed: %v", err)
	}
	if got != (models.PaymentRequest{}) {
		t.Errorf("Expected zero value, got %v", got)
	}
}

func TestQueue_EnqueueMultiple(t *testing.T) {
	ctx := context.Background()
	q := NewQueue(0)

	payments := []models.PaymentRequest{
		{Amount: decimal.New(1, 0)},
		{Amount: decimal.New(2, 0)},
		{Amount: decimal.New(3, 0)},
	}
	for _, p := range payments {
		if err := q.Enqueue(ctx, p); err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	for _, expected := range payments {
		got, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue failed: %v", err)
		}
		if got.Amount != expected.Amount {
			t.Errorf("Expected %v, got %v", expected.Amount, got.Amount)
		}
	}
}

func BenchmarkQueue_BulkEnqueue(b *testing.B) {
	ctx := context.Background()
	payment := models.PaymentRequest{Amount: decimal.New(42, 0)}
	q := NewQueue(20000)
	for n := 0; n < b.N; n++ {
		_ = q.Enqueue(ctx, payment)
	}
}

func BenchmarkQueue_BulkDequeue(b *testing.B) {
	ctx := context.Background()
	const bulk = 20000
	payment := models.PaymentRequest{Amount: decimal.New(42, 0)}
	q := NewQueue(20000)
	for n := 0; n < bulk; n++ {
		_ = q.Enqueue(ctx, payment)
	}
	for n := 0; n < b.N; n++ {
		_, _ = q.Dequeue(ctx)
	}
}
