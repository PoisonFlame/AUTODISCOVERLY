package handlers

import (
	"encoding/xml"
	"net/http/httptest"
	"testing"
)

func TestAutoconfig_ByEmailAddressParam(t *testing.T) {
	handler := NewAutoconfigHandler(newTestResolver())

	req := httptest.NewRequest("GET", "/mail/config-v1.1.xml?emailaddress=user@example.com", nil)
	req.Host = "autoconfig.example.com"
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	var cfg autoconfigClientConfig
	if err := xml.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if cfg.Provider.Domain != "example.com" {
		t.Errorf("Provider.Domain = %q, want example.com", cfg.Provider.Domain)
	}
	if cfg.Provider.IncomingServer.Hostname != "mail.mxrouting.net" || cfg.Provider.IncomingServer.Port != 993 {
		t.Errorf("unexpected incomingServer: %+v", cfg.Provider.IncomingServer)
	}
	if cfg.Provider.IncomingServer.Username != "user@example.com" {
		t.Errorf("incomingServer.Username = %q, want full email", cfg.Provider.IncomingServer.Username)
	}
	if cfg.Provider.OutgoingServer.Hostname != "mail.mxrouting.net" || cfg.Provider.OutgoingServer.Port != 465 {
		t.Errorf("unexpected outgoingServer: %+v", cfg.Provider.OutgoingServer)
	}
}

func TestAutoconfig_WellKnownPathUsesHostHeader(t *testing.T) {
	handler := NewAutoconfigHandler(newTestResolver())

	req := httptest.NewRequest("GET", "/.well-known/autoconfig/mail/config-v1.1.xml", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	var cfg autoconfigClientConfig
	if err := xml.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if cfg.Provider.Domain != "example.com" {
		t.Errorf("Provider.Domain = %q, want example.com", cfg.Provider.Domain)
	}
	// No emailaddress param was given, so the placeholder should be preserved.
	if cfg.Provider.IncomingServer.Username != "%EMAILADDRESS%" {
		t.Errorf("incomingServer.Username = %q, want placeholder", cfg.Provider.IncomingServer.Username)
	}
}

func TestAutoconfig_AutoconfigHostPrefixStripped(t *testing.T) {
	handler := NewAutoconfigHandler(newTestResolver())

	req := httptest.NewRequest("GET", "/mail/config-v1.1.xml", nil)
	req.Host = "autoconfig.example.com:443"
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
}

func TestAutoconfig_UnknownDomain(t *testing.T) {
	handler := NewAutoconfigHandler(newTestResolver())

	req := httptest.NewRequest("GET", "/mail/config-v1.1.xml?emailaddress=user@unknown.test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404 for unknown domain", rec.Code)
	}
}

func TestAutoconfig_STARTTLSSocketType(t *testing.T) {
	handler := NewAutoconfigHandler(newTestResolver())

	req := httptest.NewRequest("GET", "/mail/config-v1.1.xml?emailaddress=user@starttls.example", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	var cfg autoconfigClientConfig
	if err := xml.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if cfg.Provider.IncomingServer.SocketType != "STARTTLS" {
		t.Errorf("IncomingServer.SocketType = %q, want STARTTLS", cfg.Provider.IncomingServer.SocketType)
	}
}
