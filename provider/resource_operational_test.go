package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

// TestAccMaintenance covers a one-time maintenance window over a host group,
// including tag filters, and exercises update by extending the period.
func TestAccMaintenance(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccMaintenanceConfig(id, "3600", "with_data"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "name", "acc-mnt-"+id),
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "maintenance_type", "with_data"),
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "timeperiod.#", "1"),
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "timeperiod.0.period", "3600"),
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "groups.#", "1"),
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "tag.#", "1"),
				),
			},
			{
				// longer period and no data collection, which also drops the tags
				Config: testAccMaintenanceConfig(id, "7200", "no_data"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "timeperiod.0.period", "7200"),
					resource.TestCheckResourceAttr("zabbix_maintenance.m", "maintenance_type", "no_data"),
				),
			},
		},
	})
}

func testAccMaintenanceConfig(id, period, mtype string) string {
	tags := ""
	if mtype == "with_data" {
		tags = `
  tag {
    key      = "service"
    value    = "web"
    operator = "2"
  }
`
	}
	return fmt.Sprintf(`
resource "zabbix_hostgroup" "g" {
  name = "acc-mnt-grp-%s"
}

resource "zabbix_maintenance" "m" {
  name             = "acc-mnt-%s"
  description      = "created by the acceptance test"
  maintenance_type = "%s"
  active_since     = "1800000000"
  active_till      = "1800086400"
  groups           = [zabbix_hostgroup.g.id]
%s
  timeperiod {
    type       = "one_time"
    start_date = "1800000000"
    period     = "%s"
  }
}
`, id, id, mtype, tags, period)
}

// TestAccMaintenanceRecurring covers a weekly recurring period, which uses a
// different subset of the timeperiod fields.
func TestAccMaintenanceRecurring(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "g" {
  name = "acc-mntw-grp-%s"
}

resource "zabbix_maintenance" "w" {
  name         = "acc-mntw-%s"
  active_since = "1800000000"
  active_till  = "1830000000"
  groups       = [zabbix_hostgroup.g.id]

  timeperiod {
    type       = "weekly"
    every      = "1"
    dayofweek  = "64"
    start_time = "3600"
    period     = "7200"
  }
}
`, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_maintenance.w", "timeperiod.0.type", "weekly"),
					resource.TestCheckResourceAttr("zabbix_maintenance.w", "timeperiod.0.dayofweek", "64"),
					resource.TestCheckResourceAttr("zabbix_maintenance.w", "timeperiod.0.start_time", "3600"),
				),
			},
		},
	})
}

// TestAccProxygroup covers the Zabbix 7.0 proxy group resource.
func TestAccProxygroup(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_proxygroup" "pg" {
  name           = "acc-pg-%s"
  description    = "created by the acceptance test"
  failover_delay = "1m"
  min_online     = "1"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxygroup.pg", "name", "acc-pg-"+id),
					resource.TestCheckResourceAttr("zabbix_proxygroup.pg", "failover_delay", "1m"),
					resource.TestCheckResourceAttrSet("zabbix_proxygroup.pg", "id"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_proxygroup" "pg" {
  name           = "acc-pg-renamed-%s"
  description    = "updated"
  failover_delay = "5m"
  min_online     = "2"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxygroup.pg", "name", "acc-pg-renamed-"+id),
					resource.TestCheckResourceAttr("zabbix_proxygroup.pg", "failover_delay", "5m"),
					resource.TestCheckResourceAttr("zabbix_proxygroup.pg", "min_online", "2"),
				),
			},
		},
	})
}

// TestAccProxyActive covers an active mode proxy, the common case.
func TestAccProxyActive(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_proxy" "p" {
  name           = "acc-proxy-%s"
  operating_mode = "active"
  description    = "created by the acceptance test"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.p", "name", "acc-proxy-"+id),
					resource.TestCheckResourceAttr("zabbix_proxy.p", "operating_mode", "active"),
					resource.TestCheckResourceAttrSet("zabbix_proxy.p", "id"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_proxy" "p" {
  name              = "acc-proxy-renamed-%s"
  operating_mode    = "active"
  description       = "updated"
  allowed_addresses = "127.0.0.1"
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.p", "name", "acc-proxy-renamed-"+id),
					resource.TestCheckResourceAttr("zabbix_proxy.p", "allowed_addresses", "127.0.0.1"),
				),
			},
		},
	})
}

// TestAccProxyPassiveInGroup covers a passive proxy that is a member of a proxy
// group, which requires local_address to be set.
func TestAccProxyPassiveInGroup(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_proxygroup" "pg" {
  name           = "acc-pxg-%s"
  failover_delay = "1m"
  min_online     = "1"
}

resource "zabbix_proxy" "p" {
  name           = "acc-pxp-%s"
  operating_mode = "passive"
  address        = "127.0.0.1"
  port           = "10051"
  proxy_groupid  = zabbix_proxygroup.pg.id
  local_address  = "127.0.0.1"
  local_port     = "10051"
}
`, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_proxy.p", "operating_mode", "passive"),
					resource.TestCheckResourceAttr("zabbix_proxy.p", "address", "127.0.0.1"),
					resource.TestCheckResourceAttrPair(
						"zabbix_proxy.p", "proxy_groupid", "zabbix_proxygroup.pg", "id"),
				),
			},
		},
	})
}
