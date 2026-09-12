package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

var payPeriods = []string{"weekly", "biweekly", "semimonthly", "monthly"}

type config struct {
	Port           string
	BasePath       string
	Namespace      string
	PaymentsAPIKey string
	PayPeriod      string
	DailyRateCents int
	Kafka          kafkaConfig
}

type kafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
	Username      string
	Password      string
	Mechanism     string
	TLS           bool
}

func loadConfig() config {
	cfg := config{
		Port:           envOr("PORT", "8080"),
		BasePath:       normalizeBasePath(os.Getenv("BASE_PATH")),
		Namespace:      envOr("NAMESPACE", "local"),
		PaymentsAPIKey: os.Getenv("PAYMENTS_API_KEY"),
		PayPeriod:      envOr("PAY_PERIOD", "biweekly"),
		DailyRateCents: envInt("DEMO_DAILY_RATE_CENTS", 32000),
		Kafka: kafkaConfig{
			Brokers:       splitCSV(os.Getenv("KAFKA_BROKERS")),
			Topic:         os.Getenv("KAFKA_TOPIC"),
			ConsumerGroup: os.Getenv("KAFKA_CONSUMER_GROUP"),
			Username:      os.Getenv("KAFKA_USERNAME"),
			Password:      os.Getenv("KAFKA_PASSWORD"),
			Mechanism:     envOr("KAFKA_SASL_MECHANISM", "SCRAM-SHA-512"),
		},
	}
	cfg.Kafka.TLS = envBool("KAFKA_TLS", cfg.Kafka.Username != "")

	if cfg.PaymentsAPIKey == "" {
		log.Fatal("PAYMENTS_API_KEY is required")
	}
	if len(cfg.Kafka.Brokers) == 0 || cfg.Kafka.Topic == "" || cfg.Kafka.ConsumerGroup == "" {
		log.Fatal("KAFKA_BROKERS, KAFKA_TOPIC, and KAFKA_CONSUMER_GROUP are required")
	}
	valid := false
	for _, p := range payPeriods {
		valid = valid || p == cfg.PayPeriod
	}
	if !valid {
		log.Fatalf("PAY_PERIOD must be one of %s, got %q", strings.Join(payPeriods, ", "), cfg.PayPeriod)
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
