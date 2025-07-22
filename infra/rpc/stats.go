package rpc

import (
	"github.com/shopspring/decimal"
	"github.com/valyala/gorpc"
	"niltonkummer/rinha-2025/pkg/models"
	"time"
)

type rpcStatsAdapter struct {
	rpcClient *gorpc.Client
}

func NewStats(rpcClient *gorpc.Client) *rpcStatsAdapter {
	return &rpcStatsAdapter{
		rpcClient: rpcClient,
	}
}

func (s *rpcStatsAdapter) InsertStats(correlationID string, amount decimal.Decimal, fallback bool, requestedAt time.Time) error {

	_, err := s.rpcClient.Call(&models.StatsRPC{
		CorrelationID: correlationID,
		Amount:        amount,
		Fallback:      fallback,
		CreatedAt:     requestedAt,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *rpcStatsAdapter) RetrieveTotalStatsByPeriod(fallback bool, start, end time.Time) (*models.PaymentsSummaryResponse, error) {

	response, err := s.rpcClient.Call(&models.SummaryRPC{
		Start: start,
		End:   end,
	})
	if err != nil {
		return nil, err
	}

	return response.(*models.PaymentsSummaryResponse), nil
}
