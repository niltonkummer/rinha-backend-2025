package stats

import (
	"github.com/gofiber/fiber/v2/log"
	"github.com/shopspring/decimal"
	"niltonkummer/rinha-2025/pkg/adapters"
	"niltonkummer/rinha-2025/pkg/models"
	"strconv"
	"sync"
	"time"
)

//
/*type PaymentsSummary struct {
	TotalRequests int             `json:"totalRequests"`
	TotalAmount   decimal.Decimal `json:"totalAmount"`
}

type PaymentsSummaryResponse struct {
	Default  PaymentsSummary `json:"default"`
	Fallback PaymentsSummary `json:"fallback"`
}*/

// StatsService provides methods to manage and retrieve statistics
// This service must count atomically the number of requests and the total amount of money processed
type StatsService struct {
	// Add fields as needed for the stats service
	isFallback    bool // Indicates if this is a fallback stats service
	TotalRequests int
	TotalAmount   decimal.Decimal
	mutex         *sync.RWMutex
	storage       adapters.StatsAdapter
}

// NewStatsService creates a new instance of StatsService
func NewStatsService(isFallback bool, storage adapters.StatsAdapter) *StatsService {
	return &StatsService{
		isFallback:    isFallback,
		TotalRequests: 0,
		TotalAmount:   decimal.NewFromInt(0),
		mutex:         &sync.RWMutex{},
		storage:       storage, // Initialize with a nil storage adapter, can be set later
	}
}

// IncrementRequest increments the total number of requests
func (s *StatsService) IncrementRequest(payment models.PaymentRequest) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.TotalRequests++
	s.TotalAmount = s.TotalAmount.Add(payment.Amount)

	// Store the stats in the database
	if err := s.storage.InsertStats(payment.CorrelationID, payment.Amount, s.isFallback, payment.RequestedAt); err != nil {
		// Handle the error, e.g., log it or return it
		// For now, we will just ignore it
		log.Error("Failed to insert stats", "error", err)
		return
	}
}

// GetStats returns the current statistics
func (s *StatsService) GetStatsByPeriod(start, end time.Time) *models.PaymentsSummaryResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sum, err := s.storage.RetrieveTotalStatsByPeriod(s.isFallback, start, end)
	if err != nil {
		// Handle the error, e.g., log it or return it
		// For now, we will just ignore it
		// log.Error("Failed to retrieve stats by period", "error", err)
		return nil
	}

	return sum
}

func (s *StatsService) ResetStats() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.TotalRequests = 0
	s.TotalAmount = decimal.NewFromInt(0)
}

func (s *StatsService) PrintStats() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return "Total Requests: " + strconv.Itoa(s.TotalRequests) + ", Total Amount: " + s.TotalAmount.String()
}
