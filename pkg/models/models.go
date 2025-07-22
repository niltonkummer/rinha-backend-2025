package models

import (
	"github.com/shopspring/decimal"
	"time"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

type PaymentRequest struct {
	CorrelationID string          `json:"correlationId"`
	Amount        decimal.Decimal `json:"amount"`
	RequestedAt   time.Time       `json:"requestedAt"`
}

type HealthResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type HealthControl struct {
	DefaultProvider  HealthResponse
	FallbackProvider HealthResponse
}

type PaymentsSummary struct {
	TotalRequests int             `json:"totalRequests"`
	TotalAmount   decimal.Decimal `json:"totalAmount"`
}

type PaymentsSummaryResponse struct {
	Default  PaymentsSummary `json:"default"`
	Fallback PaymentsSummary `json:"fallback"`
}

type EnqueueRPC struct {
	Request *PaymentRequest
}

type DequeueRPC struct {
	Request *PaymentRequest
}

type StatsRPC struct {
	CorrelationID string          `json:"correlationId"`
	Amount        decimal.Decimal `json:"amount"`
	Fallback      bool            `json:"fallback"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type SummaryRPC struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
