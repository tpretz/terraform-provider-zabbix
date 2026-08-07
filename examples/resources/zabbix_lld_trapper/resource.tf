data "zabbix_host" "example" {
  host = "server-1.example.com"
}

# a trapper discovery rule is fed by zabbix_sender, so Zabbix requires delay = 0
resource "zabbix_lld_trapper" "pushed" {
  hostid = data.zabbix_host.example.id
  name   = "Pushed discovery"
  key    = "pushed.discovery"
  delay  = "0"
}
