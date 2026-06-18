package httpserver

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath      = "configs/httpserver.yaml"
	DefaultHTTPListenAddr  = ":8080"
	DefaultGRPCTarget      = "dns:///localhost:50051"
	DefaultWebDir          = "web"
	DefaultRequestTimeout  = 3 * time.Second
)

type Config struct {
	HTTP struct {
		ListenAddr string `yaml:"listen_addr"`
	} `yaml:"http"`
	GRPC struct {
		Target string `yaml:"target"`
	} `yaml:"grpc"`
	Server struct {
		RequestTimeout string `yaml:"request_timeout"`
	} `yaml:"server"`
	Web struct {
		Dir string `yaml:"dir"`
	} `yaml:"web"`
	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

func LoadConfig(path string) (*Config, error) {
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

func (c *Config) RequestTimeout() time.Duration {
	timeout, err := time.ParseDuration(c.Server.RequestTimeout)
	if err != nil {
		return DefaultRequestTimeout
	}
	return timeout
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.HTTP.ListenAddr = DefaultHTTPListenAddr
	cfg.GRPC.Target = DefaultGRPCTarget
	cfg.Server.RequestTimeout = DefaultRequestTimeout.String()
	cfg.Web.Dir = DefaultWebDir
	cfg.Logging.Level = "info"
	return cfg
}

func validateConfig(cfg *Config) error {
	if cfg.HTTP.ListenAddr == "" {
		return fmt.Errorf("http.listen_addr is required")
	}
	if cfg.GRPC.Target == "" {
		return fmt.Errorf("grpc.target is required")
	}
	if cfg.Web.Dir == "" {
		return fmt.Errorf("web.dir is required")
	}
	return nil
}
