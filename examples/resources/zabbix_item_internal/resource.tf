data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_internal" "unsupported_items" {
  hostid    = data.zabbix_host.example.id
  name      = "Unsupported items on this host"
  key       = "zabbix[host,,items_unsupported]"
  valuetype = "unsigned"
  delay     = "5m"
}
