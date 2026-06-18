package pkg

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath    = "configs/grpcserver.yaml"
	DefaultGRPCListen    = ":50051"
	DefaultRedisAddr     = "localhost:6379"
	DefaultRedisKey      = "voting:topics"
	DefaultRequestTimeout = 3 * time.Second
)

type AppConfig struct {
	GRPC struct {
		ListenAddr string `yaml:"listen_addr"`
	} `yaml:"grpc"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		Key      string `yaml:"key"`
	} `yaml:"redis"`
	Vote struct {
		Topics []string `yaml:"topics"`
	} `yaml:"vote"`
	Server struct {
		RequestTimeout string `yaml:"request_timeout"`
	} `yaml:"server"`
	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

func LoadConfig(path string) (*AppConfig, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	cfg := defaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *AppConfig) RequestTimeout() time.Duration {
	timeout, err := time.ParseDuration(c.Server.RequestTimeout)
	if err != nil {
		return DefaultRequestTimeout
	}
	return timeout
}

func defaultConfig() *AppConfig {
	cfg := &AppConfig{}
	cfg.GRPC.ListenAddr = DefaultGRPCListen
	cfg.Redis.Addr = DefaultRedisAddr
	cfg.Redis.Key = DefaultRedisKey
	cfg.Server.RequestTimeout = DefaultRequestTimeout.String()
	cfg.Vote.Topics = []string{"Golang", "Kubernetes", "Rust"}
	cfg.Logging.Level = "info"
	return cfg
}

func validateConfig(cfg *AppConfig) error {
	if cfg.GRPC.ListenAddr == "" {
		return fmt.Errorf("grpc.listen_addr is required")
	}
	if cfg.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
	}
	if cfg.Redis.Key == "" {
		return fmt.Errorf("redis.key is required")
	}
	if len(cfg.Vote.Topics) != 3 {
		return fmt.Errorf("vote.topics must contain exactly 3 items")
	}
	seen := make(map[string]struct{}, len(cfg.Vote.Topics))
	for _, topic := range cfg.Vote.Topics {
		if topic == "" {
			return fmt.Errorf("vote.topics contains empty topic")
		}
		if _, ok := seen[topic]; ok {
			return fmt.Errorf("vote.topics contains duplicate topic: %s", topic)
		}
		seen[topic] = struct{}{}
	}
	return nil
}
