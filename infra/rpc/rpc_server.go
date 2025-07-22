package rpc

import (
	"context"
	"github.com/valyala/gorpc"
	"log/slog"
	"niltonkummer/rinha-2025/pkg/models"
)

// RPCServer is a simple RPC server that handles requests
type RPCServer struct {
	log *slog.Logger
}

func NewRPCServer(logger *slog.Logger) *RPCServer {
	return &RPCServer{
		log: logger,
	}
}

func RegisterTypes(types ...any) {
	for _, t := range types {
		gorpc.RegisterType(t)
	}
}

// Start starts the RPC server
func (s *RPCServer) Start(ctx context.Context, addr string, handler func(request any) any) error {

	gorpc.RegisterType(&models.PaymentsSummary{})

	server := &gorpc.Server{
		// Accept clients on this TCP address.
		Addr: addr,

		Handler: func(clientAddr string, request interface{}) interface{} {
			return handler(request)
		},
	}

	go func() {
		<-ctx.Done()
		s.log.Info("Stopping RPC server")
		server.Stop()
	}()

	err := server.Serve()
	if err != nil {
		s.log.Error("Cannot start rpc server: %s", err)
	}
	return err
}
