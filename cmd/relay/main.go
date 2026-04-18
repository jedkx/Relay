package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"relay/internal/delivery"
	"relay/internal/httpserver"
	"relay/internal/store"
	"relay/internal/webhook"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("missing DATABASE_URL")
	}

	bg := context.Background()
	db, err := store.OpenPostgres(bg, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	workerCtx, stopWorker := context.WithCancel(bg)
	delivery.Start(workerCtx, db)

	whHandler := webhook.NewHandler(db)
	router := httpserver.NewRouter(whHandler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("relay listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down relay...")
	stopWorker()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("relay stopped")
}
