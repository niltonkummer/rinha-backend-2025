package payment

import (
	"context"
	"log/slog"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/request"
)

// Service POST payments
// GET payments-summary
type Service struct {
	Request         request.RequestService
	RequestFallback request.RequestService
	log             *slog.Logger
}

func NewPaymentService(request, request2 request.RequestService,
	log *slog.Logger) *Service {
	s := &Service{
		Request:         request,
		RequestFallback: request2,
		log:             log,
	}

	providers = map[int]*ServiceStatus{
		Default: {
			request: s.Request,
		},
		Fallback: {
			request: s.RequestFallback,
		},
	}

	s.healthChecker()
	go s.HealthChecker()
	return s
}

// PaymentRequest handles the payment request
func (s *Service) PaymentRequest(ctx context.Context, request models.PaymentRequest) (any, error) {
	requester := s.getHealthyRequest()
	response, err := requester.request.Post(ctx, "/payments", request, nil)
	if err != nil {
		requester.totalFailures.Add(1)
		requester.healthStatus.Failing = true
		return nil, err
	}
	requester.healthStatus.Failing = false
	return response, nil
}

func (s *Service) getHealthyRequest() (requester *ServiceStatus) {
	status := GetProviderHealth()
	perc := float64(status.DefaultProvider.MinResponseTime) / float64(status.FallbackProvider.MinResponseTime)

	if status.DefaultProvider.Failing || perc > 1.5 {
		currentProvider = Fallback
	} else {
		currentProvider = Default
	}

	if status.FallbackProvider.Failing && status.DefaultProvider.Failing {
		currentProvider = Default
	}

	return providers[currentProvider]
}
