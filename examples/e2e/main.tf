terraform {
  required_providers {
    zabbix = {
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}

provider "zabbix" {
  # credentials come from ZABBIX_URL / ZABBIX_USER / ZABBIX_PASS
}

# ---------------------------------------------------------------- server info

data "zabbix_server" "this" {}

# ---------------------------------------------------------------------- groups

resource "zabbix_hostgroup" "hosts" {
  name = "e2e-hosts"
}

resource "zabbix_templategroup" "templates" {
  name = "e2e-templates"
}

# -------------------------------------------------------------------- template

resource "zabbix_template" "app" {
  host        = "e2e-app-template"
  groups      = [zabbix_templategroup.templates.id]
  description = "End to end test template (updated)"

  macro {
    name  = "{$E2E_THRESHOLD}"
    value = "95"
  }
}

# ----------------------------------------------------------------------- items

resource "zabbix_item_trapper" "cpu" {
  hostid    = zabbix_template.app.id
  key       = "e2e.cpu.load"
  name      = "E2E CPU load"
  valuetype = "float"

  tag {
    key   = "component"
    value = "cpu"
  }
}

resource "zabbix_item_agent" "mem" {
  hostid    = zabbix_template.app.id
  key       = "e2e.mem.used"
  name      = "E2E memory used"
  valuetype = "unsigned"
  delay     = "5m"

  preprocessor {
    type   = "1" # multiplier
    params = ["1024"]
  }
}

resource "zabbix_item_dependent" "mem_pct" {
  hostid        = zabbix_template.app.id
  key           = "e2e.mem.pct"
  name          = "E2E memory percent"
  valuetype     = "float"
  master_itemid = zabbix_item_agent.mem.id
}

# --------------------------------------------------------------------- trigger

resource "zabbix_trigger" "cpu_high" {
  name       = "E2E CPU load too high"
  expression = "last(/${zabbix_template.app.host}/${zabbix_item_trapper.cpu.key})>${1}"
  priority   = "disaster"
  comments   = "Raised by the end to end test"

  tag {
    key   = "scope"
    value = "performance"
  }
}

# ----------------------------------------------------------------------- graph

resource "zabbix_graph" "cpu" {
  name   = "E2E CPU graph"
  height = "400"
  width  = "900"

  item {
    itemid = zabbix_item_trapper.cpu.id
    color  = "2774A4"
  }

  item {
    itemid = zabbix_item_agent.mem.id
    color  = "F63100"
  }
}

# ------------------------------------------------------------------------- LLD

resource "zabbix_lld_trapper" "fs" {
  hostid   = zabbix_template.app.id
  key      = "e2e.fs.discovery"
  name     = "E2E filesystem discovery"
  lifetime = "60d"

  condition {
    macro = "{#FSNAME}"
    value = "^/"
  }
}

resource "zabbix_proto_item_trapper" "fs_size" {
  hostid    = zabbix_template.app.id
  ruleid    = zabbix_lld_trapper.fs.id
  key       = "e2e.fs.size[{#FSNAME}]"
  name      = "E2E size of {#FSNAME}"
  valuetype = "unsigned"
}

# ------------------------------------------------------------------------ host

resource "zabbix_host" "server" {
  host      = "e2e-server"
  name      = "E2E Server"
  groups    = [zabbix_hostgroup.hosts.id]
  templates = [zabbix_template.app.id]

  interface {
    type = "agent"
    ip   = "127.0.0.1"
    port = 10050
    main = true
  }

  inventory_mode = "manual"
  inventory {
    location = "rack 43"
  }

  macro {
    name  = "{$E2E_HOST_MACRO}"
    value = "host-level-updated"
  }

  tag {
    key   = "env"
    value = "e2e"
  }
}

# --------------------------------------------------------------- data sources

data "zabbix_hostgroup" "lookup" {
  name = zabbix_hostgroup.hosts.name
}

data "zabbix_templategroup" "lookup" {
  name = zabbix_templategroup.templates.name
}

data "zabbix_template" "lookup" {
  host = zabbix_template.app.host
}

data "zabbix_host" "lookup" {
  host = zabbix_host.server.host
}

# ----------------------------------------------------------------------- output

output "server_version" {
  value = data.zabbix_server.this.version
}

output "host_id" {
  value = zabbix_host.server.id
}

output "template_id" {
  value = zabbix_template.app.id
}

output "hostgroup_lookup_matches" {
  value = data.zabbix_hostgroup.lookup.id == zabbix_hostgroup.hosts.id
}

output "templategroup_lookup_matches" {
  value = data.zabbix_templategroup.lookup.id == zabbix_templategroup.templates.id
}

output "template_lookup_matches" {
  value = data.zabbix_template.lookup.id == zabbix_template.app.id
}

output "host_lookup_matches" {
  value = data.zabbix_host.lookup.id == zabbix_host.server.id
}
