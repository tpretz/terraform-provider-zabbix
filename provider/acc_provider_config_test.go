package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// E4 -- provider configuration (PLAN.md Phase 8).
//
// Three provider-block options had no coverage at all. They are not
// interchangeable with the username/password path the rest of the suite uses:
//
//   - token skips Login() entirely and assigns api.Auth directly, which is
//     the input to the branch Phase 2a rewrote. From 6.4 the client sends the
//     token as an "Authorization: Bearer" header and from 7.2 it *must*,
//     because the "auth" body property was removed; below 6.4 it goes in the
//     body. So one token configuration exercises two different wire formats
//     across the matrix, and until now neither was exercised at all.
//   - tls_insecure replaces the http.Client wholesale with one carrying a
//     custom Transport, and every acceptance server in the matrix is plain
//     HTTP, so nothing in the suite had ever constructed that client.
//   - serialize takes a non-reentrant sync.Mutex around every request. The
//     failure mode is not a wrong answer, it is a hang.

// testAccAdminAPI builds an admin-authenticated client straight from the
// environment, for the setup a test has to do before the provider exists.
func testAccAdminAPI(t *testing.T) *zabbix.API {
	t.Helper()

	api, err := zabbix.NewAPI(zabbix.Config{Url: os.Getenv("ZABBIX_URL")})
	if err != nil {
		t.Fatalf("connecting to Zabbix: %s", err)
	}
	if _, err := api.Login(os.Getenv("ZABBIX_USER"), os.Getenv("ZABBIX_PASS")); err != nil {
		t.Fatalf("logging in to Zabbix: %s", err)
	}
	return api
}

// testAccCreateToken mints a real API token for the configured user and
// registers its removal. Zabbix returns the token string exactly once, from
// token.generate; there is no way to read it back afterwards, which is why
// this has to happen before the TestCase is built rather than in a PreConfig.
func testAccCreateToken(t *testing.T, api *zabbix.API, name string) string {
	t.Helper()

	var users []struct {
		UserID string `json:"userid"`
	}
	err := api.CallWithErrorParse("user.get", zabbix.Params{
		"output": []string{"userid"},
		"filter": map[string]interface{}{"username": os.Getenv("ZABBIX_USER")},
	}, &users)
	if err != nil {
		t.Fatalf("looking up the test user: %s", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected exactly one user named %q, got %d", os.Getenv("ZABBIX_USER"), len(users))
	}

	// a leftover token from an aborted run would collide on name
	var existing []struct {
		TokenID string `json:"tokenid"`
	}
	if err := api.CallWithErrorParse("token.get", zabbix.Params{
		"output": []string{"tokenid"},
		"filter": map[string]interface{}{"name": name},
	}, &existing); err == nil && len(existing) > 0 {
		ids := make([]string, 0, len(existing))
		for _, e := range existing {
			ids = append(ids, e.TokenID)
		}
		_, _ = api.CallWithError("token.delete", ids)
	}

	var created struct {
		TokenIDs []string `json:"tokenids"`
	}
	err = api.CallWithErrorParse("token.create", zabbix.Params{
		"name":   name,
		"userid": users[0].UserID,
	}, &created)
	if err != nil {
		t.Fatalf("creating an API token: %s", err)
	}
	if len(created.TokenIDs) != 1 {
		t.Fatalf("token.create returned %d ids", len(created.TokenIDs))
	}
	tokenID := created.TokenIDs[0]

	t.Cleanup(func() {
		if _, err := api.CallWithError("token.delete", []string{tokenID}); err != nil {
			t.Logf("could not remove test token %s: %s", tokenID, err)
		}
	})

	var generated []struct {
		Token string `json:"token"`
	}
	err = api.CallWithErrorParse("token.generate", []string{tokenID}, &generated)
	if err != nil {
		t.Fatalf("generating an API token: %s", err)
	}
	if len(generated) != 1 || generated[0].Token == "" {
		t.Fatalf("token.generate returned nothing usable: %+v", generated)
	}
	return generated[0].Token
}

// TestAccProviderTokenAuth configures the provider with a real API token and
// no username or password at all, then does a full create/read/update/destroy
// through it.
//
// Clearing ZABBIX_USER and ZABBIX_PASS for the duration is the point of the
// test, not housekeeping: the schema's DefaultFunc reads them, and with them
// still set the provider would be handed credentials it could fall back on.
// Only with them gone does a passing run prove the token was what
// authenticated every call.
func TestAccProviderTokenAuth(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test: set TF_ACC=1 to run")
	}
	testAccPreCheck(t)

	api := testAccAdminAPI(t)
	token := testAccCreateToken(t, api, "test-provider-token")
	zabbixURL := os.Getenv("ZABBIX_URL")

	t.Setenv("ZABBIX_USER", "")
	t.Setenv("ZABBIX_USERNAME", "")
	t.Setenv("ZABBIX_PASS", "")
	t.Setenv("ZABBIX_PASSWORD", "")

	config := func(name string) string {
		return fmt.Sprintf(`
provider "zabbix" {
	url   = %q
	token = %q
}

resource "zabbix_hostgroup" "testgrp" {
	name = %q
}
`, zabbixURL, token, name)
	}

	resource.Test(t, resource.TestCase{
		// no PreCheck: testAccPreCheck insists on the credentials this test
		// has deliberately removed
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // create and read
				Config: config("test-token-group"),
				Check: resource.TestCheckResourceAttr(
					"zabbix_hostgroup.testgrp", "name", "test-token-group"),
			},
			{ // update, so the token covers a write that is not a create
				Config: config("test-token-group-renamed"),
				Check: resource.TestCheckResourceAttr(
					"zabbix_hostgroup.testgrp", "name", "test-token-group-renamed"),
			},
			{ // and an import, which is a read on a bare id
				ResourceName:      "zabbix_hostgroup.testgrp",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccProviderTokenInvalid is the other half: a token the server does not
// know must fail, and fail while talking to Zabbix rather than being quietly
// ignored in favour of some other credential. Without this, a provider that
// dropped the token on the floor and logged in with the environment's
// username and password would pass TestAccProviderTokenAuth.
func TestAccProviderTokenInvalid(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test: set TF_ACC=1 to run")
	}
	testAccPreCheck(t)

	zabbixURL := os.Getenv("ZABBIX_URL")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "zabbix" {
	url   = %q
	token = "0000000000000000000000000000000000000000000000000000000000000000"
}

resource "zabbix_hostgroup" "testgrp" {
	name = "test-token-invalid-group"
}
`, zabbixURL),
				ExpectError: regexp.MustCompile(`(?i)not authori[sz]ed|re-?login|permission`),
			},
		},
	})
}

// TestAccProviderSerialize covers the serialize option. There is nothing to
// observe about a correctly serialised run -- the results are the same -- so
// what this actually guards is the failure mode: api.ex is a plain
// sync.Mutex taken for the whole of callBytes, so any request issued while
// another is in flight on the same client deadlocks rather than erroring.
// Terraform's default parallelism is 10 and the fixture stands up more
// independent objects than that, so the apply is genuinely concurrent.
func TestAccProviderSerialize(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test: set TF_ACC=1 to run")
	}
	testAccPreCheck(t)

	zabbixURL := os.Getenv("ZABBIX_URL")

	var groups string
	for i := 0; i < 12; i++ {
		groups += fmt.Sprintf(`
resource "zabbix_hostgroup" "testgrp%d" {
	name = "test-serialize-group-%d"
}
`, i, i)
	}

	config := fmt.Sprintf(`
provider "zabbix" {
	url       = %q
	serialize = true
}
`, zabbixURL) + groups

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp0", "name", "test-serialize-group-0"),
					resource.TestCheckResourceAttr("zabbix_hostgroup.testgrp11", "name", "test-serialize-group-11"),
				),
			},
			{ // and the destroy at the end of the test is a concurrent one too
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// testAccTLSProxy fronts the Zabbix server under test with an HTTPS listener
// carrying httptest's self-signed certificate, which no system trust store
// knows. Every server in the matrix is plain HTTP, so without this there is
// no way to reach the code tls_insecure controls.
//
// The director rewrites the whole URL rather than using
// NewSingleHostReverseProxy, which joins the target's path onto the incoming
// request's and would produce /api_jsonrpc.php/api_jsonrpc.php.
func testAccTLSProxy(t *testing.T, target string) *httptest.Server {
	t.Helper()

	u, err := url.Parse(target)
	if err != nil {
		t.Fatalf("parsing ZABBIX_URL: %s", err)
	}

	srv := httptest.NewTLSServer(&httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = u.Scheme
			r.URL.Host = u.Host
			r.URL.Path = u.Path
			r.Host = u.Host
		},
	})
	t.Cleanup(srv.Close)
	return srv
}

// TestAccProviderTLSInsecure drives the provider at an HTTPS endpoint whose
// certificate does not verify, and requires the two settings of tls_insecure
// to disagree about it. The negative half matters as much as the positive:
// tls_insecure defaults to false, and a client that skipped verification
// unconditionally would pass the positive half on its own.
func TestAccProviderTLSInsecure(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test: set TF_ACC=1 to run")
	}
	testAccPreCheck(t)

	proxy := testAccTLSProxy(t, os.Getenv("ZABBIX_URL"))
	proxyURL := proxy.URL + "/api_jsonrpc.php"

	config := func(insecure bool) string {
		return fmt.Sprintf(`
provider "zabbix" {
	url          = %q
	tls_insecure = %t
}

resource "zabbix_hostgroup" "testgrp" {
	name = "test-tls-group"
}
`, proxyURL, insecure)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // verification on: the untrusted certificate must stop it, and
				// it must stop it in NewAPI's version probe, before any
				// credential is sent anywhere
				Config:      config(false),
				ExpectError: regexp.MustCompile(`(?i)x509|certificate`),
			},
			{ // verification off: the same endpoint now works end to end
				Config: config(true),
				Check: resource.TestCheckResourceAttr(
					"zabbix_hostgroup.testgrp", "name", "test-tls-group"),
			},
		},
	})
}
