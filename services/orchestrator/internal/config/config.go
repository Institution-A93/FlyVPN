// Package config — конфигурация orchestrator из окружения.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr     string        // admin/HTTP API
	DatabaseURL    string        // DSN PostgreSQL
	HealthInterval time.Duration // период health-проверок
	HealthThreshold int          // подряд-неудач до пометки down
}

func FromEnv() (Config, error) {
	c := Config{
		ListenAddr:      getenv("ORCH_LISTEN", ":9090"),
		DatabaseURL:     os.Getenv("ORCH_DATABASE_URL"),
		HealthInterval:  getdur("ORCH_HEALTH_INTERVAL", 30*time.Second),
		HealthThreshold: getint("ORCH_HEALTH_THRESHOLD", 3),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("ORCH_DATABASE_URL не задан")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func getint(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func getdur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
