package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestV2JSON_KnownDomain(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@example.com&Protocol=IMAP", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	var resp v2Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Server != "mail.mxrouting.net" || resp.Port != 993 || !resp.SSL {
		t.Errorf("unexpected response: %+v", resp)
	}
	if resp.Username != "user@example.com" {
		t.Errorf("Username = %q, want full email", resp.Username)
	}
}

func TestV2JSON_DefaultsToIMAPProtocol(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@example.com", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	var resp v2Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Protocol != "IMAP" {
		t.Errorf("Protocol = %q, want IMAP default", resp.Protocol)
	}
}

func TestV2JSON_UnsupportedProtocol(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@example.com&Protocol=ActiveSync", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404 for unsupported protocol", rec.Code)
	}
}

func TestV2JSON_UnknownDomain(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@unknown.test", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404 for unknown domain", rec.Code)
	}
}

func TestV2JSON_MissingEmail(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 for missing Email", rec.Code)
	}
}
