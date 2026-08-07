data "zabbix_template" "linux" {
  host = "Linux by Zabbix agent"
}

resource "zabbix_host" "example" {
  host      = "server-1.example.com"
  groups    = [data.zabbix_hostgroup.linux_servers.id]
  templates = [data.zabbix_template.linux.id]

  interface {
    type = "agent"
    ip   = "10.0.0.10"
  }
}

data "zabbix_hostgroup" "linux_servers" {
  name = "Linux servers/production"
}
