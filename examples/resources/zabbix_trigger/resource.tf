data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_agent" "cpu_load" {
  hostid    = data.zabbix_host.example.id
  name      = "CPU load (1m average)"
  key       = "system.cpu.load[all,avg1]"
  valuetype = "float"
}

resource "zabbix_trigger" "cpu_high" {
  name       = "High CPU load on {HOST.NAME}"
  expression = "last(/server-1.example.com/system.cpu.load[all,avg1])>5"
  priority   = "high"
  comments   = "Sustained load above 5. See the capacity runbook."
  url        = "https://runbooks.example.com/cpu"

  manual_close = true

  tag {
    key   = "scope"
    value = "performance"
  }

  depends_on = [zabbix_item_agent.cpu_load]
}
