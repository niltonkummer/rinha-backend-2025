package main

import (
	"context"
	"fmt"
	"github.com/valyala/gorpc"
	"log/slog"
	config2 "niltonkummer/rinha-2025/infra/config"
	"niltonkummer/rinha-2025/infra/http"
	"niltonkummer/rinha-2025/infra/rpc"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/cache"
	"niltonkummer/rinha-2025/pkg/services/consumer"
	"niltonkummer/rinha-2025/pkg/services/handler"
	"niltonkummer/rinha-2025/pkg/services/orch"
	"niltonkummer/rinha-2025/pkg/services/payment"
	"niltonkummer/rinha-2025/pkg/services/request"
	stats2 "niltonkummer/rinha-2025/pkg/services/stats"
	"os"
	"time"
)

func api() {
	// Create Viper config instance
	config := config2.LoadConfig()

	//ctx := context.TODO()
	//ps, err := pubsub.NewRabbitMQPublisher(ctx, config.PubsubURL, "payments")
	//if err != nil {
	//	panic(err)
	//}

	// queue := redis.NewRedisQueue(config.RedisURL)

	rpcClient := gorpc.NewTCPClient(config.RPCAddr)
	rpcClient.Start()
	queue := rpc.NewPayments(rpcClient)

	log := slog.Default()
	statsAdapter := rpc.NewStats(rpcClient) //db.NewStatsAdapter(config.DatabaseURLPostgres, log)
	statsDefault := stats2.NewStatsService(false, statsAdapter)
	statsFallback := stats2.NewStatsService(true, statsAdapter)

	paymentHandler := handler.NewPaymentHandler(nil, queue, statsDefault, statsFallback, log)
	router := http.NewRouter(paymentHandler)

	router.RegisterRoutes()
	// Start the HTTP server.env
	err := router.App.Listen(":" + config.HTTPServerHost)

	if err != nil {
		fmt.Println(err)
		//panic(err)
	}
}

// Create a HTTP server.env with
func consumerProcessing() {
	// Create Viper config instance
	config := config2.LoadConfig()

	ctx := context.TODO()
	//ps, err := pubsub.NewRabbitMQPublisher(ctx, config.PubsubURL, "payments")
	//if err != nil {
	//	panic(err)
	//}

	// queue := redis.NewRedisQueue(config.RedisURL)

	rpcClient := gorpc.NewTCPClient(config.RPCAddr)
	rpcClient.Start()
	queue := rpc.NewPayments(rpcClient)

	log := slog.Default()
	// statsAdapter, err := db.NewStatsAdapter(config.DatabaseURLPostgres, log)
	statsAdapter := rpc.NewStats(rpcClient)
	statsDefault := stats2.NewStatsService(false, statsAdapter)
	statsFallback := stats2.NewStatsService(true, statsAdapter)

	go func() {
		for ; ; time.Sleep(10 * time.Second) {
			log.Info("Default: " + statsDefault.PrintStats())
			log.Info("Fallback: " + statsFallback.PrintStats())
		}
	}()

	fn := func(sts *stats2.StatsService) func(any, any) {
		return func(body any, resp any) {
			req := body.(models.PaymentRequest)
			sts.IncrementRequest(req)
		}
	}

	jobProcessor := orch.NewJobProcessor(config.MaxJobs, config.JobTimeout, log)
	paymentClientDefault := request.NewRequestService(config.ProcessorDefaultURL,
		request.WithAfterRequestFunc(fn(statsDefault)),
	)
	paymentClientFallback := request.NewRequestService(config.ProcessorFallbackURL,
		request.WithAfterRequestFunc(fn(statsFallback)))
	paymentService := payment.NewPaymentService(paymentClientDefault, paymentClientFallback, log)
	newConsumer := consumer.NewConsumer(nil, queue, paymentService, jobProcessor, log)
	go newConsumer.ConsumerQueue("payments")

	jobProcessor.HandleSteps(ctx, 1)

}

func cacheService() {
	config := config2.LoadConfig()
	log := slog.Default()

	cacheAdapter := cache.NewCacheService(20000)
	queue := cache.NewQueue(20000)
	cacheHandler := handler.NewCacheHandler(cacheAdapter, queue, slog.Default())

	server := rpc.NewRPCServer(log)
	server.Start(context.TODO(), config.RPCAddr, cacheHandler.HandleRPC)
}

func main() {
	if os.Args == nil || len(os.Args) < 2 {
		fmt.Println("Usage: rinha-2025 <mode>")
		fmt.Println("Modes: api, consumer")
		return
	}

	rpc.RegisterTypes(
		models.PaymentRequest{},
		&models.StatsRPC{},
		&models.SummaryRPC{},
		&models.PaymentsSummaryResponse{},
		&models.DequeueRPC{},
		&models.EnqueueRPC{},
	)

	mode := os.Args[1]
	switch mode {
	case "api":
		api()
	case "consumer":
		consumerProcessing()
	case "cache":
		cacheService()
	}
}
