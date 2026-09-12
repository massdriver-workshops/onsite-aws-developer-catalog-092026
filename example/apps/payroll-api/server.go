package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type server struct {
	cfg     config
	ledger  *Ledger
	started time.Time
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /info", s.info)
	mux.HandleFunc("GET /ledger", s.ledgerView)
	mux.HandleFunc("GET /events", s.events)
	return logRequests(mux)
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) info(w http.ResponseWriter, _ *http.Request) {
	consumed, last := s.ledger.Stats()
	var lastEvent any
	if !last.IsZero() {
		lastEvent = last.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service":         "payroll-api",
		"version":         version,
		"namespace":       s.cfg.Namespace,
		"pay_period":      s.cfg.PayPeriod,
		"topic":           s.cfg.Kafka.Topic,
		"consumer_group":  s.cfg.Kafka.ConsumerGroup,
		"events_consumed": consumed,
		"last_event_at":   lastEvent,
		"started_at":      s.started.UTC().Format(time.RFC3339),
		"payments_provider": map[string]any{
			"connected": true,
			"key_hint":  keyHint(s.cfg.PaymentsAPIKey),
		},
	})
}

func (s *server) ledgerView(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.ledger.View(time.Now().UTC()))
}

func (s *server) events(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.ledger.Recent())
}

// keyHint shows enough of the secret to prove it arrived and nothing more.
func keyHint(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "…" + key[len(key)-4:]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
