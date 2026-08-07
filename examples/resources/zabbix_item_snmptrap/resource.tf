data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_snmptrap" "link_traps" {
  hostid    = data.zabbix_host.example.id
  name      = "Link state traps"
  key       = "snmptrap[\"linkDown|linkUp\"]"
  valuetype = "log"
}
