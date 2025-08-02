package main

import (
	"context"
	"fmt"
	"github.com/valyala/gorpc"
	"log/slog"
	"os"
	"time"

	"github.com/jamiealquiza/tachymeter"

	configPkg "niltonkummer/rinha-2025/infra/config"
	"niltonkummer/rinha-2025/infra/http"
	"niltonkummer/rinha-2025/infra/rpc"
	"niltonkummer/rinha-2025/pkg/models"
	"niltonkummer/rinha-2025/pkg/services/cache"
	"niltonkummer/rinha-2025/pkg/services/consumer"
	"niltonkummer/rinha-2025/pkg/services/handler"
	"niltonkummer/rinha-2025/pkg/services/orch"
	"niltonkummer/rinha-2025/pkg/services/payment"
	"niltonkummer/rinha-2025/pkg/services/request"
	statsPkg "niltonkummer/rinha-2025/pkg/services/stats"
)

var (
	config = configPkg.LoadConfig()
	log    *slog.Logger
)

func api(ctx context.Context) {

	rpcClient := gorpc.NewUnixClient(config.RPCAddr)
	rpcClient.Start()

	queue := rpc.NewPayments(rpcClient)

	statsAdapter := rpc.NewStats(rpcClient)
	statsDefault := statsPkg.NewStatsService(false, statsAdapter)
	statsFallback := statsPkg.NewStatsService(true, statsAdapter)

	paymentHandler := handler.NewPaymentHandler(nil, queue, statsDefault, statsFallback, log)
	router := http.NewRouter(paymentHandler)

	router.RegisterRoutes()

	// Start the HTTP
	err := router.Start(config.ApiName)

	if err != nil {
		log.Error("error on start api service", "err", err)
	}
}

func consumerProcessing(ctx context.Context) {

	rpcClient := gorpc.NewUnixClient(config.RPCAddr)
	rpcClient.Start()
	queue := rpc.NewPayments(rpcClient)

	statsAdapter := rpc.NewStats(rpcClient)
	statsDefault := statsPkg.NewStatsService(false, statsAdapter)
	statsFallback := statsPkg.NewStatsService(true, statsAdapter)

	go func() {
		for ; ; time.Sleep(10 * time.Second) {
			log.Debug("Default: " + statsDefault.PrintStats())
			log.Debug("Fallback: " + statsFallback.PrintStats())
		}
	}()

	reqDefaultStats := statsPkg.RequestStats{
		Meter: tachymeter.New(&tachymeter.Config{Size: 15000}),
	}
	reqFallbackStats := statsPkg.RequestStats{
		Meter: tachymeter.New(&tachymeter.Config{Size: 15000}),
	}

	_ = func() {
		for ; ; time.Sleep(10 * time.Second) {
			fmt.Println("Default",
				"success", reqDefaultStats.Success.Load(),
				"fail", reqDefaultStats.Fail.Load(),
				"avg", reqDefaultStats.Meter.Calc(),
			)
			fmt.Println("Fallback",
				"success", reqFallbackStats.Success.Load(),
				"fail", reqFallbackStats.Fail.Load(),
				"avg", reqFallbackStats.Meter.Calc(),
			)
		}
	}

	fn := func(sts *statsPkg.StatsService) func(any, any) {
		return func(body any, resp any) {
			req := body.(models.PaymentRequest)
			sts.IncrementRequest(req)
		}
	}

	fnReqStats := func(sts *statsPkg.RequestStats) func(success bool, duration time.Duration) {
		return func(success bool, duration time.Duration) {
			if success {
				sts.Success.Add(1)
			} else {
				sts.Fail.Add(1)
			}
			sts.Meter.AddTime(duration)
		}
	}

	paymentClientDefault := request.NewRequestService(
		config.ProcessorDefaultURL,
		request.WithAfterSuccessRequestFunc(fn(statsDefault)),
		request.WithAfterRequestFunc(fnReqStats(&reqDefaultStats)),
	)
	paymentClientFallback := request.NewRequestService(
		config.ProcessorFallbackURL,
		request.WithAfterSuccessRequestFunc(fn(statsFallback)),
		request.WithAfterRequestFunc(fnReqStats(&reqFallbackStats)),
	)
	paymentService := payment.NewPaymentService(paymentClientDefault, paymentClientFallback, log)

	jobProcessor := orch.NewJobProcessor(config.MaxJobs, config.JobTimeout, log)

	newConsumer := consumer.NewConsumer(nil, queue, paymentService, jobProcessor, log)

	go newConsumer.ConsumerQueue()

	err := jobProcessor.HandleSteps(ctx, 2)
	if err != nil {
		log.Error("error on start HandleSteps", "err", err)
		return
	}

}

func cacheService(ctx context.Context) {

	cacheAdapter := cache.NewCacheService(16000)
	queue := cache.NewQueue(16000)
	cacheHandler := handler.NewCacheHandler(cacheAdapter, queue, slog.Default())

	server := rpc.NewRPCServer(log)

	err := server.Start(ctx, config.RPCAddr, cacheHandler.HandleRPC)
	if err != nil {
		log.Error("error on start cache service", "err", err)
	}
}

func init() {

	level := slog.LevelInfo
	if config.Debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	log = slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func main() {
	if os.Args == nil || len(os.Args) < 2 {
		fmt.Println("Usage: rinha-2025 <mode>")
		fmt.Println("Modes: api, consumer")
		return
	}

	ctx := context.TODO()

	mode := os.Args[1]

	rpc.RegisterTypes(
		models.PaymentRequest{},
		&models.StatsRPC{},
		&models.SummaryRPC{},
		&models.PaymentsSummaryResponse{},
		&models.DequeueRPC{},
		models.EnqueueRPC{},
		models.DequeueBatchRPC{},
	)

	switch mode {
	case "api":
		api(ctx)
	case "consumer":
		consumerProcessing(ctx)
	case "cache":
		cacheService(ctx)
	}
}
