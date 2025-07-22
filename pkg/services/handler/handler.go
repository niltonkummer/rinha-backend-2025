package handler

import (
	"context"
	"fmt"
	"log/slog"
	"niltonkummer/rinha-2025/infra/pubsub"
	"niltonkummer/rinha-2025/pkg/adapters"
	"niltonkummer/rinha-2025/pkg/models"
	stats2 "niltonkummer/rinha-2025/pkg/services/stats"
	"time"
)

type PaymentHandler struct {
	//pubsub        pubsub.Publisher
	queue         adapters.QueueAdapter
	cache         adapters.Cache
	stats         *stats2.StatsService
	statsFallback *stats2.StatsService
	log           *slog.Logger
}

func NewPaymentHandler(publisher pubsub.Publisher, queue adapters.QueueAdapter, stats, statsFallback *stats2.StatsService, log *slog.Logger) *PaymentHandler {
	return &PaymentHandler{
		//pubsub:        publisher,
		queue:         queue,
		stats:         stats,
		statsFallback: statsFallback,
		log:           log,
	}
}

// HandlePaymentRequest processes the payment request and publishes it to the message queue
func (h *PaymentHandler) HandlePaymentRequest(ctx context.Context, request models.PaymentRequest) error {
	/*data, err := json.Marshal(request)
	if err != nil {
		return err
	}*/

	/*
		if err := h.pubsub.PublishMessage("payment.pending", "payments", data); err != nil {
			return err
		}
	*/
	// if err := h.queue.Enqueue(ctx, "payments.pending", string(data)); err != nil {
	return h.queue.Enqueue(ctx, &request)
}

// HandlePaymentsSummary processes the payments summary request and publishes it to the message queue
func (h *PaymentHandler) HandlePaymentsSummary(start, end time.Time) (*models.PaymentsSummaryResponse, error) {

	return h.stats.GetStatsByPeriod(start, end), nil
}

type CacheHandler struct {
	cache adapters.Cache
	queue adapters.QueueAdapter
	log   *slog.Logger
}

func NewCacheHandler(cache adapters.Cache, queue adapters.QueueAdapter, log *slog.Logger) *CacheHandler {
	return &CacheHandler{
		cache: cache,
		queue: queue,
		log:   log,
	}
}

// HandleRPC processes RPC requests for payment handling
func (h *CacheHandler) HandleRPC(request any) any {
	switch req := request.(type) {
	case *models.EnqueueRPC:
		if err := h.EnqueuePaymentRequest(context.Background(), req.Request); err != nil {
			h.log.Error("Failed to enqueue payment request", "error", err)
		}
		h.log.Debug("Enqueued payment request", "req", fmt.Sprintf("%T", req), "request", req)
		return req
	case *models.DequeueRPC:
		dequeuedRequest, err := h.DequeuePaymentRequest(context.Background())
		if err != nil {
			h.log.Error("Failed to dequeue payment request", "error", err)
			return nil
		}
		req.Request = dequeuedRequest
		h.log.Debug("Dequeued payment request", "req", fmt.Sprintf("%T", dequeuedRequest), "request", dequeuedRequest)
		return req
	case *models.StatsRPC:
		h.log.Debug("Received StatsRPC request", "req", fmt.Sprintf("%T", req), "request", req)
		h.AddStat(req)
		return request

	case *models.SummaryRPC:
		summary, err := h.GetStats(req.Start, req.End)
		if err != nil {
			h.log.Error("Failed to retrieve stats", "error", err)
			return nil
		}
		return summary
	}

	return request
}

func (h *CacheHandler) EnqueuePaymentRequest(ctx context.Context, request *models.PaymentRequest) error {
	return h.queue.Enqueue(ctx, request)
}

func (h *CacheHandler) DequeuePaymentRequest(ctx context.Context) (*models.PaymentRequest, error) {
	request, err := h.queue.Dequeue(ctx)
	if err != nil {
		h.log.Error("Failed to dequeue payment request", "error", err)
		return nil, err
	}

	if request == nil {
		return nil, nil
	}

	return request.(*models.PaymentRequest), nil
}

func (h *CacheHandler) AddStat(stats *models.StatsRPC) {
	h.cache.Add(stats)
}

func (h *CacheHandler) GetStats(start, end time.Time) (*models.PaymentsSummaryResponse, error) {
	list, err := h.cache.GetList()
	if err != nil {
		h.log.Error("Failed to retrieve stats from cache", "error", err)
		return nil, err
	}
	defaultSummary := models.PaymentsSummary{}
	fallbackSummary := models.PaymentsSummary{}
	now := time.Now()
	defer func() {
		h.log.Info("Stats summary processed", "default_requests", defaultSummary.TotalRequests, "default_amount", defaultSummary.TotalAmount,
			"fallback_requests", fallbackSummary.TotalRequests, "fallback_amount", fallbackSummary.TotalAmount, "duration", time.Since(now))
	}()
	for _, item := range list {
		if stat, ok := item.(*models.StatsRPC); ok {
			if stat.CreatedAt.After(start) && stat.CreatedAt.Before(end) {
				if stat.Fallback {
					fallbackSummary.TotalRequests++
					fallbackSummary.TotalAmount = fallbackSummary.TotalAmount.Add(stat.Amount)
				} else {
					defaultSummary.TotalRequests++
					defaultSummary.TotalAmount = defaultSummary.TotalAmount.Add(stat.Amount)
				}
			}
		}
	}
	summaryResponse := &models.PaymentsSummaryResponse{
		Default:  defaultSummary,
		Fallback: fallbackSummary,
	}
	return summaryResponse, nil
}
