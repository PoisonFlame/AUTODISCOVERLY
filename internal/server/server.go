// Package server wires the HTTP routes to their handlers.
package server

import (
	"net/http"

	"autodiscoverly/internal/handlers"
	"autodiscoverly/internal/mailconfig"
)

// New builds the top-level HTTP handler for the service.
func New(resolver *mailconfig.Resolver) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /autodiscover/autodiscover.xml", handlers.NewAutodiscoverXMLHandler(resolver))
	mux.HandleFunc("GET /autodiscover/autodiscover.json", handlers.NewAutodiscoverV2Handler(resolver))
	mux.HandleFunc("GET /mail/config-v1.1.xml", handlers.NewAutoconfigHandler(resolver))
	mux.HandleFunc("GET /.well-known/autoconfig/mail/config-v1.1.xml", handlers.NewAutoconfigHandler(resolver))
	mux.HandleFunc("GET /health", handlers.Health)

	return mux
}
