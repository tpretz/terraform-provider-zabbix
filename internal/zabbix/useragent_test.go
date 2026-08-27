package zabbix

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUserAgent pins that the caller's identification reaches the wire. The
// provider sets this to carry its own version, so a Zabbix access log records
// which provider build made a call — the first thing worth knowing when a
// report says "it broke after upgrading".
func TestUserAgent(t *testing.T) {
	for _, tc := range []struct{ name, configured, want string }{
		{"caller supplies one", "terraform-provider-zabbix/2.0.0", "terraform-provider-zabbix/2.0.0"},
		{"caller supplies none", "", "github.com/tpretz/terraform-provider-zabbix"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("User-Agent")
				fmt.Fprint(w, `{"jsonrpc":"2.0","result":"7.4.13","id":1}`)
			}))
			defer srv.Close()

			if _, err := NewAPI(Config{Url: srv.URL, UserAgent: tc.configured}); err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("User-Agent = %q, want %q", got, tc.want)
			}
		})
	}
}
