data "zabbix_host" "example" {
  host = "server-1.example.com"
}

resource "zabbix_item_agent" "cpu_load" {
  hostid    = data.zabbix_host.example.id
  name      = "CPU load (1m average)"
  key       = "system.cpu.load[all,avg1]"
  valuetype = "float"
}

# Build the expression from references rather than repeating the host and key as
# literals. Terraform sees zabbix_item_agent.cpu_load in the interpolation and
# orders the trigger after the item on its own, so no depends_on is needed, and
# editing the item's key updates the expression instead of leaving the trigger
# pointing at an item that no longer exists.
#
# Address the host by its technical name -- data.zabbix_host.example.host, not
# .name. The display name is not what a trigger expression resolves against.
resource "zabbix_trigger" "cpu_high" {
  name       = "High CPU load on {HOST.NAME}"
  expression = "last(/${data.zabbix_host.example.host}/${zabbix_item_agent.cpu_load.key})>5"
  priority   = "high"
  comments   = "Sustained load above 5. See the capacity runbook."
  url        = "https://runbooks.example.com/cpu"

  manual_close = true

  tag {
    key   = "scope"
    value = "performance"
  }
}

# recovery_expression is built the same way, so a trigger that recovers on a
# different threshold still carries no duplicated literals.
resource "zabbix_trigger" "cpu_high_with_recovery" {
  name                = "High CPU load (hysteresis) on {HOST.NAME}"
  expression          = "last(/${data.zabbix_host.example.host}/${zabbix_item_agent.cpu_load.key})>5"
  recovery_expression = "last(/${data.zabbix_host.example.host}/${zabbix_item_agent.cpu_load.key})<3"
  priority            = "high"
}
