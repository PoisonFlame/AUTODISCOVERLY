package handlers

import (
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
)

const soapGetUserSettingsRequest = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:a="http://schemas.microsoft.com/exchange/2010/Autodiscover">
  <soap:Body>
    <a:GetUserSettingsRequestMessage>
      <a:Request>
        <a:Users>
          <a:User>
            <a:Mailbox>user@example.com</a:Mailbox>
          </a:User>
        </a:Users>
      </a:Request>
    </a:GetUserSettingsRequestMessage>
  </soap:Body>
</soap:Envelope>`

// The response is marshaled with literal "soap:" prefixed element names (to
// match real-world SOAP output) rather than proper namespace-URI tags, so
// Go's namespace-aware Unmarshal can't target the same soapEnvelope struct
// used to build it. This mirror type matches on local element name only,
// which is namespace-agnostic and good enough to assert on the values.
type soapTestEnvelope struct {
	Body struct {
		ResponseMessage struct {
			Response soapResponse `xml:"Response"`
		} `xml:"GetUserSettingsResponseMessage"`
	} `xml:"Body"`
}

func doSOAPRequest(t *testing.T, body string) soapResponse {
	t.Helper()
	handler := NewAutodiscoverXMLHandler(newTestResolver())

	req := httptest.NewRequest("POST", "/autodiscover/autodiscover.xml", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var env soapTestEnvelope
	if err := xml.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshaling response: %v\nbody: %s", err, rec.Body.String())
	}
	return env.Body.ResponseMessage.Response
}

func settingValue(resp soapResponse, name string) (string, bool) {
	for _, ur := range resp.UserResponses {
		for _, s := range ur.UserSettings {
			if s.Name == name {
				return s.Value, true
			}
		}
	}
	return "", false
}

func TestSOAP_DispatchedFromAutodiscoverXML(t *testing.T) {
	resp := doSOAPRequest(t, soapGetUserSettingsRequest)

	if resp.ErrorCode != "NoError" {
		t.Fatalf("ErrorCode = %q, want NoError", resp.ErrorCode)
	}

	host, ok := settingValue(resp, "InternalImapServer")
	if !ok || host != "mail.mxrouting.net" {
		t.Errorf("InternalImapServer = %q (found=%v), want mail.mxrouting.net", host, ok)
	}
	port, ok := settingValue(resp, "InternalImapPort")
	if !ok || port != "993" {
		t.Errorf("InternalImapPort = %q (found=%v), want 993", port, ok)
	}
}

func TestSOAP_UnknownDomain(t *testing.T) {
	body := strings.Replace(soapGetUserSettingsRequest, "user@example.com", "user@unknown.test", 1)
	resp := doSOAPRequest(t, body)

	if len(resp.UserResponses) == 0 || resp.UserResponses[0].ErrorCode != "InvalidUser" {
		t.Fatalf("expected InvalidUser error for unknown domain, got %+v", resp.UserResponses)
	}
}
