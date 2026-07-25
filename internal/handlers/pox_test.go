package handlers

import (
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
)

const poxOutlookRequest = `<?xml version="1.0" encoding="utf-8"?>
<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/outlook/requestschema/2006">
  <Request>
    <EMailAddress>user@example.com</EMailAddress>
    <AcceptableResponseSchema>http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a</AcceptableResponseSchema>
  </Request>
</Autodiscover>`

const poxMobileSyncRequest = `<?xml version="1.0" encoding="utf-8"?>
<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/outlook/requestschema/2006">
  <Request>
    <EMailAddress>user@example.com</EMailAddress>
    <AcceptableResponseSchema>http://schemas.microsoft.com/exchange/autodiscover/mobilesync/responseschema/2006</AcceptableResponseSchema>
  </Request>
</Autodiscover>`

func doPOXRequest(t *testing.T, body string) poxEnvelope {
	t.Helper()
	handler := NewAutodiscoverXMLHandler(newTestResolver())

	req := httptest.NewRequest("POST", "/autodiscover/autodiscover.xml", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var env poxEnvelope
	if err := xml.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshaling response: %v\nbody: %s", err, rec.Body.String())
	}
	return env
}

func TestPOX_KnownDomain(t *testing.T) {
	env := doPOXRequest(t, poxOutlookRequest)

	if env.Response.Error != nil {
		t.Fatalf("unexpected error response: %+v", env.Response.Error)
	}
	if env.Response.User == nil || env.Response.User.DisplayName != "Example Mail" {
		t.Errorf("User.DisplayName = %+v, want Example Mail", env.Response.User)
	}
	if env.Response.Account == nil || len(env.Response.Account.Protocol) != 2 {
		t.Fatalf("expected 2 protocols, got %+v", env.Response.Account)
	}

	var imap, smtp *poxProtocol
	for i := range env.Response.Account.Protocol {
		p := &env.Response.Account.Protocol[i]
		switch p.Type {
		case "IMAP":
			imap = p
		case "SMTP":
			smtp = p
		}
	}
	if imap == nil || imap.Server != "mail.mxrouting.net" || imap.Port != 993 || imap.SSL != "on" {
		t.Errorf("unexpected IMAP protocol block: %+v", imap)
	}
	if imap.LoginName != "user@example.com" {
		t.Errorf("IMAP LoginName = %q, want full email", imap.LoginName)
	}
	if smtp == nil || smtp.Server != "mail.mxrouting.net" || smtp.Port != 465 {
		t.Errorf("unexpected SMTP protocol block: %+v", smtp)
	}
}

func TestPOX_UnknownDomain(t *testing.T) {
	body := strings.Replace(poxOutlookRequest, "user@example.com", "user@unknown.test", 1)
	env := doPOXRequest(t, body)

	if env.Response.Error == nil {
		t.Fatal("expected an Error element for an unknown domain")
	}
	if env.Response.Account != nil {
		t.Errorf("unexpected Account block on an error response: %+v", env.Response.Account)
	}
}

func TestPOX_MobileSyncRequest_RespondsWithError(t *testing.T) {
	env := doPOXRequest(t, poxMobileSyncRequest)

	if env.Response.Error == nil {
		t.Fatal("expected an Error element for a mobilesync (ActiveSync) request, we don't support it")
	}
}

func TestPOX_MalformedBody(t *testing.T) {
	env := doPOXRequest(t, "not xml at all")

	if env.Response.Error == nil {
		t.Fatal("expected an Error element for a malformed request body")
	}
}

func TestPOX_STARTTLSDomain(t *testing.T) {
	body := strings.Replace(poxOutlookRequest, "user@example.com", "user@starttls.example", 1)
	env := doPOXRequest(t, body)

	if env.Response.Error != nil {
		t.Fatalf("unexpected error: %+v", env.Response.Error)
	}
	for _, p := range env.Response.Account.Protocol {
		if p.Type == "IMAP" && (p.Port != 143 || p.SSL != "off") {
			t.Errorf("STARTTLS domain IMAP protocol = %+v, want port 143 SSL off", p)
		}
	}
}
