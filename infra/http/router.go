package http

import (
	"net/http"
	"sync"
	"time"

	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/handler"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

type Router struct {
	App     *fiber.App
	Handler *handler.PaymentHandler
}

func NewRouter(paymentHandler *handler.PaymentHandler) *Router {

	app := fiber.New(fiber.Config{
		//Prefork:                  true,
		DisableHeaderNormalizing: true,
		JSONEncoder:              json.Marshal,
		JSONDecoder:              json.Unmarshal,
	})

	return &Router{
		App:     app,
		Handler: paymentHandler,
	}
}

// RegisterRoutes registers the HTTP routes for the application
func (r *Router) RegisterRoutes() {

	r.App.Use(func(c *fiber.Ctx) error {

		// Middleware to calculate request processing time
		start := time.Now()
		err := c.Next()
		if err != nil {
			// If an error occurs, log it and return a 500 status code
			c.Status(http.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"error": "Internal Server Error: " + err.Error(),
			})
		}
		// Call the next handler in the chain
		duration := time.Now().Sub(start)
		// Log the request processing time
		c.Response().Header.Set("X-Processing-Time", duration.String())
		return nil
	})
	// Health check route
	r.App.Get("/health", r.HealthCheck)

	// Payment request route
	r.App.Post("/payments", r.PaymentRequest)

	// Payments summary route
	r.App.Get("/payments-summary", r.PaymentsSummary)
}

// HealthCheck handles the health check endpoint
func (r *Router) HealthCheck(c *fiber.Ctx) error {
	// Simulate a health check response
	response := map[string]interface{}{
		"status":  "ok",
		"message": "Service is running",
	}

	return c.JSON(response)
}

var paymentPool = sync.Pool{}

func init() {
	// Initialize the payment pool with a function to create new payment objects
	paymentPool.New = func() any {
		return &models.PaymentRequest{}
	}
}

func getPaymentFromPool() *models.PaymentRequest {
	return paymentPool.Get().(*models.PaymentRequest)
}

// PaymentRequest handles the payment request endpoint
func (r *Router) PaymentRequest(c *fiber.Ctx) error {
	// Simulate processing a payment request
	var request = getPaymentFromPool()
	defer paymentPool.Put(request)

	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	err := r.Handler.HandlePaymentRequest(c.Context(), *request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process payment request: " + err.Error(),
		})
	}
	/**/
	return c.Status(http.StatusNoContent).Send(nil)
}

// PaymentsSummary handles the payments summary endpoint
func (r *Router) PaymentsSummary(c *fiber.Ctx) error {
	// GET /payments-summary?from=2020-07-10T12:34:56.000Z&to=2020-07-10T12:35:56.000Z
	type QueryParser struct {
		From string `query:"from"`
		To   string `query:"to"`
	}

	// Parse the query parameters
	var query QueryParser
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters: " + err.Error(),
		})
	}
	// Validate the date format
	from, err := time.Parse(time.RFC3339, query.From)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid 'from' date format: " + err.Error(),
		})
	}
	to, err := time.Parse(time.RFC3339, query.To)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid 'to' date format: " + err.Error(),
		})
	}

	// Simulate fetching payments summary
	summary, err := r.Handler.HandlePaymentsSummary(from.UTC(), to.UTC())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch payments summary: " + err.Error(),
		})
	}

	return c.JSON(summary)
}

// Start starts the HTTP server.env
func (r *Router) Start(port string) error {
	return r.App.Listen(port)
}
