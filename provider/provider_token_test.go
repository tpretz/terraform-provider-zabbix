package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

// clearCredentialEnv removes every credential environment variable the provider
// reads, restoring them when the test finishes.
func clearCredentialEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"ZABBIX_USER", "ZABBIX_USERNAME", "ZABBIX_PASS", "ZABBIX_PASSWORD",
		"ZABBIX_API_TOKEN", "ZABBIX_TOKEN",
	} {
		if old, ok := os.LookupEnv(k); ok {
			os.Unsetenv(k)
			key := k
			val := old
			t.Cleanup(func() { os.Setenv(key, val) })
		}
	}
}

// TestProviderRequiresCredentials verifies the provider refuses to configure
// itself when neither an API token nor a username/password pair is supplied.
func TestProviderRequiresCredentials(t *testing.T) {
	clearCredentialEnv(t)

	p := Provider()
	err := p.Configure(terraform.NewResourceConfigRaw(map[string]interface{}{
		"url": "http://localhost/api_jsonrpc.php",
	}))

	if err == nil {
		t.Fatal("expected configuration to fail without credentials, got nil")
	}
	if !strings.Contains(err.Error(), "no credentials provided") {
		t.Fatalf("expected a credentials error, got: %s", err)
	}
}

// TestProviderRejectsPasswordWithoutUsername checks the partial-credentials case.
func TestProviderRejectsPasswordWithoutUsername(t *testing.T) {
	clearCredentialEnv(t)

	p := Provider()
	err := p.Configure(terraform.NewResourceConfigRaw(map[string]interface{}{
		"url":      "http://localhost/api_jsonrpc.php",
		"password": "secret",
	}))

	if err == nil {
		t.Fatal("expected configuration to fail with a password but no username")
	}
	if !strings.Contains(err.Error(), "no credentials provided") {
		t.Fatalf("expected a credentials error, got: %s", err)
	}
}

// TestAccToken covers the zabbix_token resource lifecycle, including that the
// generated secret is exported and that the token actually authenticates.
func TestAccToken(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_token" "t" {
  name        = "acc-token-` + id + `"
  description = "created by the acceptance test"
  enabled     = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_token.t", "name", "acc-token-"+id),
					resource.TestCheckResourceAttr("zabbix_token.t", "enabled", "true"),
					resource.TestCheckResourceAttrSet("zabbix_token.t", "token"),
					resource.TestCheckResourceAttrSet("zabbix_token.t", "userid"),
					// the secret must actually work against the API
					testAccCheckTokenAuthenticates("zabbix_token.t"),
				),
			},
			{
				// disable and rename, exercising update
				Config: `
resource "zabbix_token" "t" {
  name        = "acc-token-renamed-` + id + `"
  description = "updated by the acceptance test"
  enabled     = false
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_token.t", "name", "acc-token-renamed-"+id),
					resource.TestCheckResourceAttr("zabbix_token.t", "enabled", "false"),
					// a disabled token must be refused
					testAccCheckTokenRejected("zabbix_token.t"),
				),
			},
		},
	})
}

// testAccCheckTokenAuthenticates asserts the token stored in state can be used
// to authenticate against Zabbix. This is the MFA-safe code path.
func testAccCheckTokenAuthenticates(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, err := tokenClientFromState(s, name)
		if err != nil {
			return err
		}
		if err := api.CheckAuthentication(); err != nil {
			return fmt.Errorf("expected token to authenticate, got: %s", err)
		}
		if api.Auth != "" {
			return fmt.Errorf("token client unexpectedly holds a session id")
		}
		return nil
	}
}

// testAccCheckTokenRejected asserts the token is refused (e.g. once disabled).
func testAccCheckTokenRejected(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, err := tokenClientFromState(s, name)
		if err != nil {
			return err
		}
		if err := api.CheckAuthentication(); err == nil {
			return fmt.Errorf("expected disabled token to be rejected, but it was accepted")
		}
		return nil
	}
}

func tokenClientFromState(s *terraform.State, name string) (*zabbix.API, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok {
		return nil, fmt.Errorf("%s not found in state", name)
	}
	secret := rs.Primary.Attributes["token"]
	if secret == "" {
		return nil, fmt.Errorf("%s has no token attribute", name)
	}
	return zabbix.NewAPI(zabbix.Config{
		Url:      os.Getenv("ZABBIX_URL"),
		ApiToken: secret,
	})
}

// TestAccApiTokenAuthEndToEnd configures a provider instance with nothing but an
// API token and performs a real write through it, proving the provider never
// needs user.login. This is the path required when the account uses MFA.
func TestAccApiTokenAuthEndToEnd(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC to run acceptance tests")
	}
	testAccPreCheck(t)

	url := os.Getenv("ZABBIX_URL")

	// Create a token out of band using username/password auth.
	admin, err := zabbix.NewAPI(zabbix.Config{Url: url})
	if err != nil {
		t.Fatalf("connecting to zabbix: %s", err)
	}
	if _, err := admin.Login(os.Getenv("ZABBIX_USER"), os.Getenv("ZABBIX_PASS")); err != nil {
		t.Fatalf("login: %s", err)
	}

	name := "acc-e2e-token-" + resource.UniqueId()
	created, err := admin.TokensCreate(zabbix.Tokens{{Name: name}})
	if err != nil {
		t.Fatalf("creating token: %s", err)
	}
	defer admin.TokensDeleteByIds([]string{created[0].TokenID})

	secrets, err := admin.TokensGenerate([]string{created[0].TokenID})
	if err != nil {
		t.Fatalf("generating token secret: %s", err)
	}

	// Configure a provider using only that token.
	clearCredentialEnv(t)
	p := Provider()
	if err := p.Configure(terraform.NewResourceConfigRaw(map[string]interface{}{
		"url":       url,
		"api_token": secrets[0].Token,
	})); err != nil {
		t.Fatalf("configuring provider with api token: %s", err)
	}

	tokenAPI, ok := p.Meta().(*zabbix.API)
	if !ok {
		t.Fatal("provider meta is not a *zabbix.API")
	}
	if tokenAPI.ApiToken == "" {
		t.Fatal("provider did not retain the api token")
	}
	if tokenAPI.Auth != "" {
		t.Fatal("provider performed a session login despite an api token being set")
	}

	// Exercise a real write through the token authenticated client.
	groups := zabbix.HostGroups{{Name: name}}
	if err := tokenAPI.HostGroupsCreate(groups); err != nil {
		t.Fatalf("creating hostgroup via token auth: %s", err)
	}
	defer tokenAPI.HostGroupsDeleteByIds([]string{groups[0].GroupID})

	read, err := tokenAPI.HostGroupGetByID(groups[0].GroupID)
	if err != nil {
		t.Fatalf("reading hostgroup via token auth: %s", err)
	}
	if read.Name != name {
		t.Fatalf("expected hostgroup %q, got %q", name, read.Name)
	}
	t.Logf("token auth verified against Zabbix %s", tokenAPI.Config.VersionString)
}

// TestTokenImportRefused asserts that zabbix_token declares an importer which
// refuses the operation, rather than silently importing a token with no secret.
func TestTokenImportRefused(t *testing.T) {
	r := resourceToken()

	if r.Importer == nil {
		t.Fatal("zabbix_token has no importer; terraform would report a generic " +
			"'does not support import' message instead of explaining why")
	}

	d := r.Data(&terraform.InstanceState{ID: "42"})
	res, err := r.Importer.State(d, nil)

	if err == nil {
		t.Fatal("expected zabbix_token import to be refused, got no error")
	}
	if res != nil {
		t.Fatalf("expected no imported resources, got %d", len(res))
	}
	for _, want := range []string{
		"does not support import",
		"only once",
		"data.zabbix_token",
		"42",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("import refusal message should mention %q, got: %s", want, err)
		}
	}
}

// TestAccDataToken covers the metadata-only data source, including that it does
// not expose a secret.
func TestAccDataToken(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
resource "zabbix_token" "t" {
  name        = "acc-dstoken-` + id + `"
  description = "looked up by the data source test"
  enabled     = true
}

data "zabbix_token" "found" {
  name   = zabbix_token.t.name
  userid = zabbix_token.t.userid
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.zabbix_token.found", "id", "zabbix_token.t", "id"),
					resource.TestCheckResourceAttr(
						"data.zabbix_token.found", "name", "acc-dstoken-"+id),
					resource.TestCheckResourceAttr(
						"data.zabbix_token.found", "description", "looked up by the data source test"),
					resource.TestCheckResourceAttr("data.zabbix_token.found", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.zabbix_token.found", "created_at"),
					// the data source must not carry a secret at all
					testAccCheckNoTokenSecret("data.zabbix_token.found"),
				),
			},
		},
	})
}

// testAccCheckNoTokenSecret asserts the given resource holds no token secret.
func testAccCheckNoTokenSecret(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("%s not found in state", name)
		}
		if v, present := rs.Primary.Attributes["token"]; present && v != "" {
			return fmt.Errorf("%s exposed a token secret, which it must never do", name)
		}
		return nil
	}
}

// TestDataTokenHasNoSecretAttribute is a schema level guard: the data source
// must not even declare a "token" attribute, so a secret cannot leak into state.
func TestDataTokenHasNoSecretAttribute(t *testing.T) {
	if _, present := dataToken().Schema["token"]; present {
		t.Fatal("data.zabbix_token declares a \"token\" attribute; the secret must not be " +
			"exposed through a data source")
	}
}
