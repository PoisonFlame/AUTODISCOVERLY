// Command autodiscoverly serves Outlook Autodiscover and Mozilla Autoconfig
// responses for the mail domains listed in its YAML config file.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"autodiscoverly/internal/config"
	"autodiscoverly/internal/mailconfig"
	"autodiscoverly/internal/server"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "GET the /health endpoint and exit 0/1 accordingly, for use as a Docker HEALTHCHECK (distroless images have no shell/curl to do this any other way)")
	flag.Parse()

	if *healthcheck {
		os.Exit(runHealthcheck())
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/autodiscoverly/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	resolver := mailconfig.New(cfg)
	if err := resolver.ValidateAll(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           server.New(resolver),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownComplete := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
		close(shutdownComplete)
	}()

	log.Printf("autodiscoverly listening on %s (%d domains configured)", cfg.Server.Listen, len(cfg.Domains))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	<-shutdownComplete
}

// runHealthcheck backs the -healthcheck flag: it GETs the /health endpoint
// and returns a process exit code, so Docker's HEALTHCHECK can shell out to
// this same binary instead of needing curl/wget, neither of which exist in
// the distroless image.
func runHealthcheck() int {
	addr := os.Getenv("HEALTHCHECK_ADDR")
	if addr == "" {
		addr = "http://127.0.0.1:8080/health"
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(addr)
	if err != nil {
		log.Printf("healthcheck request failed: %v", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("healthcheck got status %d", resp.StatusCode)
		return 1
	}
	return 0
}
