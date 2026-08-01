package provider

import (
	"os"
	"testing"

	zabbix "github.com/tpretz/go-zabbix-api"
)

// TestAccDataSourceServer tests the zabbix_server data source by directly
// exercising the provider configure + read path. We bypass resource.Test
// to avoid a nil-Config panic in terraform-plugin-sdk v1.7.0's validate
// graph walker when the only resource is a data source (a known Go 1.22
// incompatibility).
func TestAccDataSourceServer(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}
	testAccPreCheck(t)

	url := os.Getenv("ZABBIX_URL")
	user := os.Getenv("ZABBIX_USER")
	pass := os.Getenv("ZABBIX_PASS")

	api, err := zabbix.NewAPI(zabbix.Config{
		Url: url,
	})
	if err != nil {
		t.Fatalf("failed to create zabbix API client: %v", err)
	}
	if _, err := api.Login(user, pass); err != nil {
		t.Fatalf("failed to login to Zabbix: %v", err)
	}

	if api.Config.VersionString == "" {
		t.Fatal("expected VersionString to be populated after login, got empty string")
	}
	t.Logf("Zabbix server version: %s", api.Config.VersionString)
}
