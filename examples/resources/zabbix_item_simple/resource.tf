data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_simple" "icmp_ping" {
  hostid    = data.zabbix_host.example.id
  name      = "ICMP ping"
  key       = "icmpping"
  valuetype = "unsigned"
  delay     = "1m"
}
