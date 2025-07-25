package payment

import (
	"context"
	"errors"
	"log/slog"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/request"
	"time"
)

var noopService = &ServiceStatus{
	request: noop{},
}

// Service POST payments
// GET payments-summary
type Service struct {
	Request         request.RequestService
	RequestFallback request.RequestService
	RequestNoOp     request.RequestService
	log             *slog.Logger
}

func NewPaymentService(requestDefault, requestFallback request.RequestService,
	log *slog.Logger) *Service {
	s := &Service{
		Request:         requestDefault,
		RequestFallback: requestFallback,
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
		return nil, err
	}
	return response, nil
}

func (s *Service) getHealthyRequest() (requester *ServiceStatus) {
	status := providers

	if status[Default].totalFailures.Load() > 12 { // 12 * 5s = 60s outage
		return providers[Fallback]
	}

	if status[Default].healthStatus.Failing && status[Fallback].healthStatus.Failing {
		// currentProvider = Default
		return noopService
	}

	return providers[Default]
}

type noop struct {
}

func (n noop) Post(ctx context.Context, url string, body any, response any) (request.Response, error) {
	time.Sleep(time.Millisecond * 100)
	return request.Response{}, errors.New("cooldown")
}

func (n noop) Get(ctx context.Context, url string, response any) (request.Response, error) {
	return request.Response{}, errors.New("noop")
}
