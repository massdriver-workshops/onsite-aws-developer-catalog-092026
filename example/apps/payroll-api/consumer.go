package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// consume replays the topic from the beginning on every start and never commits
// offsets. The ledger lives in memory, so a restart rebuilds it from the topic
// instead of resuming from a committed offset with an empty ledger. The group
// still exists so ACLs and cluster tooling see a named consumer.
func consume(ctx context.Context, cfg kafkaConfig, ledger *Ledger) {
	dialer := &kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}
	if cfg.TLS {
		dialer.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if cfg.Username != "" {
		algo := scram.SHA512
		if cfg.Mechanism == "SCRAM-SHA-256" {
			algo = scram.SHA256
		}
		mech, err := scram.Mechanism(algo, cfg.Username, cfg.Password)
		if err != nil {
			log.Fatalf("kafka: scram: %v", err)
		}
		dialer.SASLMechanism = mech
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.ConsumerGroup,
		Topic:          cfg.Topic,
		Dialer:         dialer,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0,
		MaxWait:        time.Second,
		// Short timeouts so a replaced pod takes over the group in seconds, not the 30s default.
		SessionTimeout:    10 * time.Second,
		RebalanceTimeout:  10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		Logger:            kafka.LoggerFunc(func(string, ...any) {}),
		ErrorLogger:       kafka.LoggerFunc(func(msg string, args ...any) { log.Printf("kafka: "+msg, args...) }),
	})
	// Close leaves the group, so a graceful shutdown hands partitions to the next pod immediately.
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("kafka: close: %v", err)
		}
		log.Print("kafka: left consumer group")
	}()
	log.Printf("consuming %s as group %s", cfg.Topic, cfg.ConsumerGroup)

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("kafka: fetch: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		var ev Event
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("kafka: skipping malformed message at offset %d: %v", msg.Offset, err)
			continue
		}
		ledger.Apply(ev)
	}
}
