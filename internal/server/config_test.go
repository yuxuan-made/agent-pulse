package server_test

import (
	"testing"

	"github.com/yuxuan-made/agent-pulse/internal/server"
)

func TestValidateServeConfigRejectsNonLoopbackWithoutAuth(t *testing.T) {
	cfg := server.Config{Host: "0.0.0.0", Port: 8765}

	if err := server.ValidateConfig(cfg); err == nil {
		t.Fatal("expected non-loopback bind without auth to be rejected")
	}
}

func TestValidateServeConfigAllowsLoopbackWithoutAuth(t *testing.T) {
	cfg := server.Config{Host: "127.0.0.1", Port: 8765}

	if err := server.ValidateConfig(cfg); err != nil {
		t.Fatalf("expected loopback bind without auth to be allowed, got %v", err)
	}
}

func TestValidateServeConfigAllowsNonLoopbackWithAuth(t *testing.T) {
	cfg := server.Config{Host: "0.0.0.0", Port: 8765, AuthToken: "local-secret"}

	if err := server.ValidateConfig(cfg); err != nil {
		t.Fatalf("expected non-loopback bind with auth to be allowed, got %v", err)
	}
}
