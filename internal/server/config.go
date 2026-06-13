package server

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Config struct {
	Host         string
	Port         int
	AuthToken    string
	UnsafeNoAuth bool
}

func ValidateConfig(cfg Config) error {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port %d", cfg.Port)
	}
	if isLoopbackHost(host) {
		return nil
	}
	if cfg.AuthToken != "" || cfg.UnsafeNoAuth {
		return nil
	}
	return errors.New("refusing non-loopback bind without --auth-token or --unsafe-no-auth")
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
