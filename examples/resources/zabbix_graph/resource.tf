data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_agent" "cpu_user" {
  hostid    = data.zabbix_host.example.id
  name      = "CPU utilisation, user"
  key       = "system.cpu.util[,user]"
  valuetype = "float"
}

resource "zabbix_item_agent" "cpu_system" {
  hostid    = data.zabbix_host.example.id
  name      = "CPU utilisation, system"
  key       = "system.cpu.util[,system]"
  valuetype = "float"
}

resource "zabbix_graph" "cpu" {
  name   = "CPU utilisation"
  width  = "900"
  height = "200"
  type   = "stacked"

  # item is a set: sortorder carries the drawing order, not block position
  item {
    itemid    = zabbix_item_agent.cpu_user.id
    color     = "1A7C11"
    sortorder = "0"
  }

  item {
    itemid    = zabbix_item_agent.cpu_system.id
    color     = "F63100"
    sortorder = "1"
  }
}
