package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jchen0102/grok-build/internal/httpapi"
	"github.com/jchen0102/grok-build/internal/store"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := waitMigrate(ctx, databaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st, err := store.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(st),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func waitMigrate(ctx context.Context, databaseURL string) error {
	var last error
	for i := 0; i < 30; i++ {
		last = store.Migrate(databaseURL)
		if last == nil {
			return nil
		}
		log.Printf("migrate retry %d/30: %v", i+1, last)
		select {
		case <-ctx.Done():
			return last
		case <-time.After(time.Second):
		}
	}
	return last
}
