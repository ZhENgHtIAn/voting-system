package httpserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigSuccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "httpserver.yaml")
	content := `http:
  listen_addr: ":18080"
grpc:
  target: "dns:///127.0.0.1:50051"
server:
  request_timeout: "2s"
web:
  dir: "web"
logging:
  level: "debug"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if got, want := cfg.HTTP.ListenAddr, ":18080"; got != want {
		t.Fatalf("unexpected listen addr: got %s want %s", got, want)
	}
	if got, want := cfg.RequestTimeout(), 2*time.Second; got != want {
		t.Fatalf("unexpected timeout: got %s want %s", got, want)
	}
}

func TestLoadConfigValidationFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "httpserver.yaml")
	content := `http:
  listen_addr: ":8080"
grpc:
  target: ""
web:
  dir: "web"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

func TestRequestTimeoutFallback(t *testing.T) {
	t.Parallel()
	cfg := defaultConfig()
	cfg.Server.RequestTimeout = "invalid"
	if got, want := cfg.RequestTimeout(), DefaultRequestTimeout; got != want {
		t.Fatalf("unexpected timeout fallback: got %s want %s", got, want)
	}
}
