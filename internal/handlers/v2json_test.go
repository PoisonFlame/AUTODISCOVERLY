package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestV2JSON_AutodiscoverV1PointsBackAtPOX(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@example.com&Protocol=AutodiscoverV1", nil)
	req.Host = "autodiscover.example.com"
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	var resp v2Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Protocol != "AutodiscoverV1" {
		t.Errorf("Protocol = %q, want AutodiscoverV1", resp.Protocol)
	}
	want := "https://autodiscover.example.com/autodiscover/autodiscover.xml"
	if resp.Url != want {
		t.Errorf("Url = %q, want %q", resp.Url, want)
	}
}

func TestV2JSON_ProtocolCaseInsensitive(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@example.com&Protocol=autodiscoverv1", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200 regardless of Protocol casing", rec.Code)
	}
}

func TestV2JSON_UnsupportedProtocol(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	// Real Exchange-native protocol values that a genuine Outlook client
	// might ask for -- we don't have EWS/ActiveSync/REST endpoints, so
	// these should be rejected rather than answered with something wrong.
	for _, protocol := range []string{"ActiveSync", "Ews", "Rest", ""} {
		req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@example.com&Protocol="+protocol, nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != 404 {
			t.Errorf("Protocol=%q: status = %d, want 404", protocol, rec.Code)
		}
	}
}

func TestV2JSON_UnknownDomain(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Email=user@unknown.test&Protocol=AutodiscoverV1", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404 for unknown domain", rec.Code)
	}
}

func TestV2JSON_MissingEmail(t *testing.T) {
	handler := NewAutodiscoverV2Handler(newTestResolver())

	req := httptest.NewRequest("GET", "/autodiscover/autodiscover.json?Protocol=AutodiscoverV1", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400 for missing Email", rec.Code)
	}
}
