package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Event is the contract consumed by payroll-api. Field names are part of the
// contract; bump Version rather than renaming.
type Event struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	RequestID  int64  `json:"request_id"`
	EmployeeID int64  `json:"employee_id"`
	Employee   string `json:"employee"`
	Team       string `json:"team"`
	Kind       string `json:"kind"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Days       int    `json:"days"`
	Status     string `json:"status"`
	DecidedBy  string `json:"decided_by,omitempty"`
	Namespace  string `json:"namespace"`
	OccurredAt string `json:"occurred_at"`
}

const (
	EventRequested = "timeoff.requested"
	EventDecided   = "timeoff.decided"
)

type Publisher struct {
	writer *kafka.Writer
	topic  string
}

func newPublisher(cfg kafkaConfig) (*Publisher, error) {
	transport := &kafka.Transport{DialTimeout: 10 * time.Second}
	if cfg.TLS {
		transport.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if cfg.Username != "" {
		mech, err := scramMechanism(cfg)
		if err != nil {
			return nil, err
		}
		transport.SASL = mech
	}
	return &Publisher{
		topic: cfg.Topic,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Transport:    transport,
			WriteTimeout: 10 * time.Second,
		},
	}, nil
}

func scramMechanism(cfg kafkaConfig) (sasl.Mechanism, error) {
	algo := scram.SHA512
	if cfg.Mechanism == "SCRAM-SHA-256" {
		algo = scram.SHA256
	}
	return scram.Mechanism(algo, cfg.Username, cfg.Password)
}

// Publish keys by employee so one person's events stay ordered on a partition.
func (p *Publisher) Publish(ctx context.Context, ev Event) {
	if p == nil {
		return
	}
	body, err := json.Marshal(ev)
	if err != nil {
		log.Printf("events: marshal: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(ev.Employee),
		Value: body,
		Headers: []kafka.Header{
			{Key: "type", Value: []byte(ev.Type)},
			{Key: "namespace", Value: []byte(ev.Namespace)},
		},
	})
	if err != nil {
		log.Printf("events: publish %s for request %d: %v", ev.Type, ev.RequestID, err)
	}
}

func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}
	return p.writer.Close()
}

func eventFromRequest(typ string, r *Request, ns string) Event {
	return Event{
		Type:       typ,
		Version:    1,
		RequestID:  r.ID,
		EmployeeID: r.EmployeeID,
		Employee:   r.Employee,
		Team:       r.Team,
		Kind:       r.Kind,
		StartDate:  r.StartDate.Format("2006-01-02"),
		EndDate:    r.EndDate.Format("2006-01-02"),
		Days:       r.Days(),
		Status:     r.Status,
		DecidedBy:  r.DecidedBy.String,
		Namespace:  ns,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}
}
