data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_agent" "cpu_load" {
  hostid    = data.zabbix_host.example.id
  name      = "CPU load (1m average)"
  key       = "system.cpu.load[all,avg1]"
  valuetype = "float"
  delay     = "1m"

  tag {
    key   = "component"
    value = "cpu"
  }
}
