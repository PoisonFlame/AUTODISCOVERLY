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

// The response is marshaled with literal "soap:"/"i:" prefixed names (to
// match real-world SOAP wire output byte-for-byte) rather than proper
// namespace-URI tags, so Go's namespace-aware Unmarshal can't target the
// production structs directly: it resolves "i:type" to a namespaced
// attribute (space=xsi, local="type"), which won't match a Go tag written
// as the literal string "i:type". These mirror types match on local
// name only (namespace-agnostic), which is a correct and sufficient way to
// verify the values regardless of that asymmetry.
type testUserSetting struct {
	XSIType             string               `xml:"type,attr"`
	Name                string               `xml:"Name"`
	ProtocolConnections []protocolConnection `xml:"ProtocolConnections>ProtocolConnection"`
}

type testUserResponse struct {
	ErrorCode    string            `xml:"ErrorCode"`
	UserSettings []testUserSetting `xml:"UserSettings>UserSetting"`
}

type testSOAPResponse struct {
	ErrorCode     string             `xml:"ErrorCode"`
	UserResponses []testUserResponse `xml:"UserResponses>UserResponse"`
}

type soapTestEnvelope struct {
	Body struct {
		ResponseMessage struct {
			Response testSOAPResponse `xml:"Response"`
		} `xml:"GetUserSettingsResponseMessage"`
	} `xml:"Body"`
}

func doSOAPRequest(t *testing.T, body string) testSOAPResponse {
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

func settingConnection(resp testSOAPResponse, name string) (protocolConnection, string, bool) {
	for _, ur := range resp.UserResponses {
		for _, s := range ur.UserSettings {
			if s.Name == name && len(s.ProtocolConnections) > 0 {
				return s.ProtocolConnections[0], s.XSIType, true
			}
		}
	}
	return protocolConnection{}, "", false
}

func TestSOAP_DispatchedFromAutodiscoverXML(t *testing.T) {
	resp := doSOAPRequest(t, soapGetUserSettingsRequest)

	if resp.ErrorCode != "NoError" {
		t.Fatalf("ErrorCode = %q, want NoError", resp.ErrorCode)
	}

	conn, xsiType, ok := settingConnection(resp, "InternalImap4Connections")
	if !ok || conn.Hostname != "mail.mxrouting.net" || conn.Port != 993 || conn.EncryptionMethod != "SSL" {
		t.Errorf("InternalImap4Connections = %+v (found=%v), want mail.mxrouting.net:993 SSL", conn, ok)
	}
	if xsiType != "ProtocolConnectionCollectionSetting" {
		t.Errorf("InternalImap4Connections xsi:type = %q, want ProtocolConnectionCollectionSetting", xsiType)
	}

	smtpConn, _, ok := settingConnection(resp, "ExternalSmtpConnections")
	if !ok || smtpConn.Hostname != "mail.mxrouting.net" || smtpConn.Port != 465 {
		t.Errorf("ExternalSmtpConnections = %+v (found=%v), want mail.mxrouting.net:465", smtpConn, ok)
	}
}

func TestSOAP_UnknownDomain(t *testing.T) {
	body := strings.Replace(soapGetUserSettingsRequest, "user@example.com", "user@unknown.test", 1)
	resp := doSOAPRequest(t, body)

	if len(resp.UserResponses) == 0 || resp.UserResponses[0].ErrorCode != "InvalidUser" {
		t.Fatalf("expected InvalidUser error for unknown domain, got %+v", resp.UserResponses)
	}
}
