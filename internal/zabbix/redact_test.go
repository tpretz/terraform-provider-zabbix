package zabbix

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRedactBodyHidesSecrets is the unit half: every name in sensitiveKeys is
// hidden wherever it appears, including nested inside another object and
// inside an array element, and everything else survives so the log stays
// useful.
func TestRedactBodyHidesSecrets(t *testing.T) {
	body := []byte(`{
		"jsonrpc": "2.0",
		"method": "host.create",
		"auth": "SESSIONTOKEN",
		"params": {
			"host": "web01",
			"ipmi_password": "IPMIPW",
			"tls_psk": "DEADBEEF",
			"tls_psk_identity": "PSKID",
			"interfaces": [
				{"type": 2, "details": {"authpassphrase": "AUTHPW", "privpassphrase": "PRIVPW", "securityname": "svc"}}
			]
		}
	}`)

	got := redactBody(body)

	for _, secret := range []string{"SESSIONTOKEN", "IPMIPW", "DEADBEEF", "PSKID", "AUTHPW", "PRIVPW"} {
		if strings.Contains(got, secret) {
			t.Errorf("%q survived redaction: %s", secret, got)
		}
	}
	// the log has to stay diagnostically useful
	for _, keep := range []string{"host.create", "web01", "svc", "interfaces"} {
		if !strings.Contains(got, keep) {
			t.Errorf("%q was lost to redaction, the log is no longer useful: %s", keep, got)
		}
	}
}

// TestRedactCoversTerraformSpellings pins the gap an end-to-end check found:
// the provider logs some maps keyed by Terraform schema names rather than
// Zabbix wire names, and a passphrase escaped through the difference.
func TestRedactCoversTerraformSpellings(t *testing.T) {
	body := []byte(`{"snmp3_authpassphrase":"AUTHPW","snmp3_privpassphrase":"PRIVPW","snmp_community":"COMM","ip":"127.0.0.1"}`)

	got := redactBody(body)
	for _, secret := range []string{"AUTHPW", "PRIVPW", "COMM"} {
		if strings.Contains(got, secret) {
			t.Errorf("%q survived redaction: %s", secret, got)
		}
	}
	if !strings.Contains(got, "127.0.0.1") {
		t.Errorf("non-secret field was lost: %s", got)
	}
}

// TestRedactBodyFailsClosed pins the property that matters most: an input the
// redactor cannot parse must not be logged verbatim. The one case where
// parsing fails is the one case where nobody has checked what the bytes hold.
func TestRedactBodyFailsClosed(t *testing.T) {
	if got := redactBody([]byte(`{"password": "SECRET" ...truncated`)); strings.Contains(got, "SECRET") {
		t.Errorf("unparseable body was logged verbatim: %s", got)
	}
}

// TestRedactResponseHidesLoginToken covers the case a key name cannot express:
// user.login returns the session token as its bare "result".
func TestRedactResponseHidesLoginToken(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","result":"SESSIONTOKEN","id":2}`)

	if got := redactResponseBody("user.login", body); strings.Contains(got, "SESSIONTOKEN") {
		t.Errorf("user.login token survived redaction: %s", got)
	}
	// ... and that it is special-cased narrowly, or every list of ids vanishes
	if got := redactResponseBody("host.get", body); !strings.Contains(got, "SESSIONTOKEN") {
		t.Errorf("host.get result was redacted, which would gut the logs: %s", got)
	}
}

// TestNoSecretsReachTheLog is the end-to-end half, and the one that would have
// caught the original defect: drive a real API against a fake server and read
// back everything the logger emitted.
func TestNoSecretsReachTheLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		switch req["method"] {
		case "apiinfo.version", "APIInfo.version":
			fmt.Fprint(w, `{"jsonrpc":"2.0","result":"7.4.13","id":1}`)
		case "user.login":
			fmt.Fprint(w, `{"jsonrpc":"2.0","result":"SESSIONTOKEN","id":2}`)
		default:
			fmt.Fprint(w, `{"jsonrpc":"2.0","result":[],"id":3}`)
		}
	}))
	defer srv.Close()

	var buf strings.Builder
	api, err := NewAPI(Config{Url: srv.URL, Log: log.New(&buf, "[DEBUG] ", 0)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.Login("Admin", "SUPERSECRETPW"); err != nil {
		t.Fatal(err)
	}
	api.CallWithError("host.create", Params{
		"host":          "web01",
		"tls_psk":       "DEADBEEF",
		"ipmi_password": "IPMIPW",
	})

	out := buf.String()
	for _, secret := range []string{"SUPERSECRETPW", "SESSIONTOKEN", "DEADBEEF", "IPMIPW"} {
		if strings.Contains(out, secret) {
			t.Errorf("%q reached the log:\n%s", secret, out)
		}
	}
}
