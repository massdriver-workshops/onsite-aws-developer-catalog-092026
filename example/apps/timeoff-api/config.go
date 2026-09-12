package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type config struct {
	Port               string
	BasePath           string // "" when mounted at the root, otherwise "/<prefix>" with no trailing slash
	Namespace          string
	CompanyName        string
	DatabaseURL        string
	SessionSecret      string
	MaxDaysPerRequest  int
	AutoApprove        bool
	ApprovalWindowDays int
	Kafka              kafkaConfig
}

type kafkaConfig struct {
	Brokers   []string
	Topic     string
	Username  string
	Password  string
	Mechanism string
	TLS       bool
}

func (k kafkaConfig) enabled() bool { return len(k.Brokers) > 0 && k.Topic != "" }

func loadConfig() config {
	cfg := config{
		Port:               envOr("PORT", "8080"),
		BasePath:           normalizeBasePath(os.Getenv("BASE_PATH")),
		Namespace:          envOr("NAMESPACE", "local"),
		CompanyName:        envOr("COMPANY_NAME", "Your Company"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		SessionSecret:      os.Getenv("SESSION_SECRET"),
		MaxDaysPerRequest:  envInt("MAX_DAYS_PER_REQUEST", 30),
		AutoApprove:        envBool("AUTO_APPROVE", false),
		ApprovalWindowDays: envInt("APPROVAL_WINDOW_DAYS", 5),
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET is required")
	}

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		cfg.Kafka = kafkaConfig{
			Brokers:   splitCSV(brokers),
			Topic:     os.Getenv("KAFKA_TOPIC"),
			Username:  os.Getenv("KAFKA_USERNAME"),
			Password:  os.Getenv("KAFKA_PASSWORD"),
			Mechanism: envOr("KAFKA_SASL_MECHANISM", "SCRAM-SHA-512"),
		}
		// MSK SASL listeners are TLS-only, so credentials imply TLS unless told otherwise.
		cfg.Kafka.TLS = envBool("KAFKA_TLS", cfg.Kafka.Username != "")
	}
	return cfg
}

func normalizeBasePath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("%s: expected an integer, got %q", key, v)
	}
	return n
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Fatalf("%s: expected a boolean, got %q", key, v)
	}
	return b
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
