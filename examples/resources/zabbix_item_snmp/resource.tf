data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_snmp" "uptime" {
  hostid    = data.zabbix_host.example.id
  name      = "Device uptime"
  key       = "sysUpTime"
  snmp_oid  = "1.3.6.1.2.1.1.3.0"
  valuetype = "unsigned"
  delay     = "1m"

  # DISPLAY_STRING timeticks arrive in hundredths of a second
  preprocessor {
    type   = "multiplier"
    params = ["0.01"]
  }
}
