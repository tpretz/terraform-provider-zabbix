package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"zabbix": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testAccPreCheck(t *testing.T) {
	required := []string{"ZABBIX_URL", "ZABBIX_USER", "ZABBIX_PASS"}

	for _, envName := range required {
		if err := os.Getenv(envName); err == "" {
			t.Fatalf("environment variable %s must be set", envName)
		}
	}

	// Zabbix web containers can report healthy before the API is actually ready.
	// When the API isn't ready, apiinfo.version returns:
	//   -32603 Internal error: Unable to select configuration.
	// This wait keeps the acceptance suite from failing due to startup timing.
	waitForZabbixAPIReady(t, os.Getenv("ZABBIX_URL"))
}

func waitForZabbixAPIReady(t *testing.T, url string) {
	t.Helper()

	type rpcReq struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
		ID      int         `json:"id"`
	}
	// We'll decode loosely; we only care that "result" exists.
	var (
		// Fresh containers can take a while to initialise the DB schema before the API starts returning a version.
		// Be generous here; a slow CI runner is still a valid runner.
		timeout = 5 * time.Minute
		step    = 2 * time.Second
		start   = time.Now()
	)

	payload, err := json.Marshal(rpcReq{JSONRPC: "2.0", Method: "apiinfo.version", Params: []interface{}{}, ID: 1})
	if err != nil {
		t.Fatalf("failed to marshal apiinfo.version request: %v", err)
	}
	client := &http.Client{Timeout: 5 * time.Second}

	var lastErr error
	for time.Since(start) < timeout {
		req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			time.Sleep(step)
			continue
		}
		req.Header.Set("Content-Type", "application/json-rpc")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(step)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			lastErr = err
			time.Sleep(step)
			continue
		}

		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			time.Sleep(step)
			continue
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(body, &decoded); err != nil {
			lastErr = err
			time.Sleep(step)
			continue
		}

		// success path: result exists
		if _, ok := decoded["result"]; ok {
			return
		}

		// common not-ready error: {"error":{"data":"Unable to select configuration."}}
		lastErr = fmt.Errorf("api not ready: %s", string(body))
		time.Sleep(step)
	}

	t.Fatalf("timed out waiting for Zabbix API to become ready (%s): last error: %v", url, lastErr)
}
