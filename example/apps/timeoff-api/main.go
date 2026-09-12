package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// version is set at build time with -ldflags "-X main.version=1.0.0".
var version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := openStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	var events *Publisher
	if cfg.Kafka.enabled() {
		events, err = newPublisher(cfg.Kafka)
		if err != nil {
			log.Fatalf("kafka: %v", err)
		}
		defer events.Close()
		log.Printf("events enabled, topic %s", cfg.Kafka.Topic)
	} else {
		log.Print("events disabled: KAFKA_BROKERS or KAFKA_TOPIC not set")
	}

	api := &server{cfg: cfg, store: store, events: events}
	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", api.health) // for probes that hit the pod directly, outside the ingress prefix
	mount(root, cfg.BasePath, api.routes())

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("timeoff-api %s listening on :%s under %q for namespace %s", version, cfg.Port, cfg.BasePath+"/", cfg.Namespace)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	os.Exit(0)
}

// mount serves h under prefix, stripping the prefix so handlers see root-relative paths.
func mount(root *http.ServeMux, prefix string, h http.Handler) {
	if prefix == "" {
		root.Handle("/", h)
		return
	}
	root.Handle(prefix+"/", http.StripPrefix(prefix, h))
	root.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, prefix+"/", http.StatusPermanentRedirect)
	})
}
