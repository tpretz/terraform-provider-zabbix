data "zabbix_hostgroup" "linux_servers" {
  name = "Linux servers/production"
}

resource "zabbix_host" "example" {
  host   = "server-1.example.com"
  groups = [data.zabbix_hostgroup.linux_servers.id]

  interface {
    type = "agent"
    ip   = "10.0.0.10"
  }
}
