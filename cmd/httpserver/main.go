package main

import (
	"context"
	"fmt"
	"flag"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"example.com/voting-system/api/pb"
	httpserver "example.com/voting-system/internal/httpserver"
	"example.com/voting-system/internal/pkg/logging"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", httpserver.DefaultConfigPath, "path to httpserver config file")
	flag.Parse()

	cfg, err := httpserver.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}
	logger := logging.NewLogger("httpserver", cfg.Logging.Level)

	dialCtx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout())
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		cfg.GRPC.Target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`),
	)
	if err != nil {
		logger.Error("dial_grpc_failed", "target", cfg.GRPC.Target, "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Warn("grpc_conn_close_failed", "error", closeErr.Error())
		}
	}()

	voteClient := pb.NewVoteServiceClient(conn)
	server, err := httpserver.NewServer(voteClient, cfg.RequestTimeout(), logger)
	if err != nil {
		logger.Error("init_http_server_failed", "error", err.Error())
		os.Exit(1)
	}

	handler := server.NewHandler(cfg.Web.Dir)
	logger.Info("httpserver_started",
		"listen_addr", cfg.HTTP.ListenAddr,
		"grpc_target", cfg.GRPC.Target,
		"web_dir", cfg.Web.Dir,
		"log_level", cfg.Logging.Level,
	)
	if err := http.ListenAndServe(cfg.HTTP.ListenAddr, handler); err != nil {
		logger.Error("serve_http_failed", "error", err.Error())
		os.Exit(1)
	}
}
