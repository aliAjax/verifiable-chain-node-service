package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"verifiable-chain-node/internal/app"
)

func main() {
	addr := os.Getenv("CHAIN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	n := app.NewNode(app.DefaultConfig())
	srv := &http.Server{Addr: addr, Handler: n.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("chain node listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
