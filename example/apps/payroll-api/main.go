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

	ledger := newLedger(cfg.DailyRateCents, cfg.PayPeriod)
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		consume(ctx, cfg.Kafka, ledger)
	}()

	api := &server{cfg: cfg, ledger: ledger, started: time.Now()}
	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", api.health)
	mount(root, cfg.BasePath, api.routes())

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: root, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("payroll-api %s listening on :%s under %q for namespace %s, payments key %s", version, cfg.Port, cfg.BasePath+"/", cfg.Namespace, keyHint(cfg.PaymentsAPIKey))
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
	select {
	case <-consumerDone:
	case <-shutdownCtx.Done():
		log.Print("kafka: consumer did not leave the group in time")
	}
	os.Exit(0)
}

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
