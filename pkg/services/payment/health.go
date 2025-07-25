package payment

import (
	"context"
	"log/slog"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/request"
	"sync"
	"sync/atomic"
	"time"
)

var (
	currentProvider = Default
	providers       = map[int]*ServiceStatus{}
)

type ServiceStatus struct {
	request       request.RequestService
	totalFailures atomic.Int32
	healthStatus  models.HealthResponse
}

var (
	Default  = 0
	Fallback = 1
)

func (s *Service) HealthChecker() {

	for {
		time.Sleep(5 * time.Second)
		s.healthChecker()
	}
}

func (s *Service) healthChecker() {

	wg := &sync.WaitGroup{}
	wg.Add(len(providers))
	for name, provider := range providers {
		go func(name int, provider *ServiceStatus) {
			defer wg.Done()
			// Check if the current provider is healthy
			response := models.HealthResponse{}
			ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res, err := provider.request.Get(ctxWithTimeout, "/payments/service-health", &response)
			if err != nil {
				// If the current provider is not healthy, log the error and switch to the default provider
				provider.totalFailures.Add(1)
				return
			}

			if res.StatusCode/100 == 4 || res.StatusCode/100 == 5 {
				return
			}

			provider.healthStatus = response
		}(name, provider)
	}
	wg.Wait()
	slog.Info("Health", "default", providers[Default].healthStatus)
	slog.Info("Health", "fallback", providers[Default].healthStatus)
	return

}
