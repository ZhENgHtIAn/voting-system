package main

import (
	"context"
	"fmt"
	"flag"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"example.com/voting-system/api/pb"
	grpcserver "example.com/voting-system/internal/grpcserver"
	"example.com/voting-system/internal/pkg"
	"example.com/voting-system/internal/pkg/logging"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", pkg.DefaultConfigPath, "path to grpcserver config file")
	flag.Parse()

	cfg, err := pkg.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}
	logger := logging.NewLogger("grpcserver", cfg.Logging.Level)

	redisClient := pkg.NewRedisClient(pkg.RedisConfig{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Warn("redis_client_close_failed", "error", closeErr.Error())
		}
	}()

	pingCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout())
	defer cancel()
	if err := pkg.PingRedis(pingCtx, redisClient); err != nil {
		logger.Error("redis_not_ready", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("redis_ready", "addr", cfg.Redis.Addr)
	if err := pkg.EnsureTopicsInitialized(pingCtx, redisClient, cfg.Redis.Key, cfg.Vote.Topics); err != nil {
		logger.Error("initialize_topics_failed", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("topics_initialized", "topics", cfg.Vote.Topics)

	voteService, err := grpcserver.NewVoteService(redisClient, cfg.Redis.Key, cfg.Vote.Topics, logger)
	if err != nil {
		logger.Error("init_vote_service_failed", "error", err.Error())
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", cfg.GRPC.ListenAddr)
	if err != nil {
		logger.Error("listen_grpc_failed", "addr", cfg.GRPC.ListenAddr, "error", err.Error())
		os.Exit(1)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(unaryLoggingInterceptor(logger)))
	pb.RegisterVoteServiceServer(server, voteService)

	logger.Info("grpcserver_started", "listen_addr", cfg.GRPC.ListenAddr, "log_level", cfg.Logging.Level)
	if err := server.Serve(listener); err != nil {
		logger.Error("serve_grpc_failed", "error", err.Error())
		os.Exit(1)
	}
}

func unaryLoggingInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		requestID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("x-request-id"); len(values) > 0 {
				requestID = values[0]
			}
		}
		clientAddr := ""
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			clientAddr = p.Addr.String()
		}
		logger.Info("grpc_request_started",
			"method", info.FullMethod,
			"request_id", requestID,
			"client_addr", clientAddr,
		)

		resp, err := handler(ctx, req)
		code := status.Code(err).String()
		logger.Info("grpc_request_completed",
			"method", info.FullMethod,
			"request_id", requestID,
			"code", code,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return resp, err
	}
}
