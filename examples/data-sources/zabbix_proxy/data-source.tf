data "zabbix_proxy" "existing" {
  name = "proxy-eu-west"
}

resource "zabbix_hostgroup" "example" {
  name = "Remote sites"
}

resource "zabbix_host" "example" {
  host    = "server-1.example.com"
  groups  = [zabbix_hostgroup.example.id]
  proxyid = data.zabbix_proxy.existing.id

  interface {
    type = "agent"
    ip   = "10.0.0.10"
  }
}
