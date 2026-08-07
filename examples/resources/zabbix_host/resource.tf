resource "zabbix_hostgroup" "linux_servers" {
  name = "Linux servers/production"
}

data "zabbix_template" "linux" {
  host = "Linux by Zabbix agent"
}

resource "zabbix_host" "example" {
  host      = "server-1.example.com"
  name      = "server-1"
  groups    = [zabbix_hostgroup.linux_servers.id]
  templates = [data.zabbix_template.linux.id]

  interface {
    type = "agent"
    ip   = "10.0.0.10"
    port = 10050
  }

  interface {
    type           = "snmp"
    ip             = "10.0.0.10"
    snmp_version   = "2"
    snmp_community = "{$SNMP_COMMUNITY}"
  }

  macro {
    name  = "{$SNMP_COMMUNITY}"
    value = "public"
  }

  tag {
    key   = "env"
    value = "production"
  }

  inventory_mode = "manual"

  inventory {
    location = "dc1/rack14"
    contact  = "platform@example.com"
  }
}

# interface is a set, so it cannot be indexed: use one() or a for expression
output "agent_interface_id" {
  value = one([for i in zabbix_host.example.interface : i.id if i.type == "agent"])
}
