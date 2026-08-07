data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_calculated" "total_throughput" {
  hostid    = data.zabbix_host.example.id
  name      = "Total interface throughput"
  key       = "net.if.total"
  formula   = "last(//net.if.in)+last(//net.if.out)"
  valuetype = "float"
  delay     = "1m"
}
