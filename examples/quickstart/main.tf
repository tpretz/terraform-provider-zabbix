# Complete working example covering the most common use cases.
# All resources in this file are independent — remove what you don't need.
#
# Run it:
#   export ZABBIX_URL=http://your-zabbix/api_jsonrpc.php
#   export ZABBIX_USER=Admin ZABBIX_PASS=zabbix   # or ZABBIX_API_TOKEN
#   tofu init && tofu apply

terraform {
  required_providers {
    zabbix = {
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}

provider "zabbix" {
  # credentials from environment variables
}

# ------------------------------------------------------------------ data sources

data "zabbix_server" "this" {}

output "zabbix_version" { value = data.zabbix_server.this.version }

# ---------------------------------------------------------------- host grouping

resource "zabbix_hostgroup" "linux" {
  name = "Linux servers"
}

resource "zabbix_templategroup" "app" {
  name = "Templates/Application"
}

# -------------------------------------------------------------------- template

resource "zabbix_template" "app" {
  host        = "my-application"
  groups      = [zabbix_templategroup.app.id]
  description = "Application monitoring template"

  macro {
    name  = "{$APP_PORT}"
    value = "8080"
  }
}

# ----------------------------------------------------------------- items

resource "zabbix_item_agent" "cpu" {
  hostid    = zabbix_template.app.id
  key       = "system.cpu.load[all,avg1]"
  name      = "CPU load (1 min avg)"
  valuetype = "float"
  delay     = "1m"
}

resource "zabbix_item_trapper" "deploy_ts" {
  hostid    = zabbix_template.app.id
  key       = "deploy.timestamp"
  name      = "Last deployment timestamp"
  valuetype = "unsigned"
}

# ----------------------------------------------------------------- trigger

resource "zabbix_trigger" "cpu_high" {
  name       = "High CPU load on {HOST.NAME}"
  expression = "avg(/${zabbix_template.app.host}/${zabbix_item_agent.cpu.key},5m)>4"
  priority   = "warn"
  comments   = "CPU load averaged over 5 minutes exceeded 4"

  tag {
    key   = "component"
    value = "cpu"
  }
}

# ----------------------------------------------------------------- LLD

resource "zabbix_lld_agent" "fs" {
  hostid   = zabbix_template.app.id
  key      = "vfs.fs.discovery"
  name     = "Filesystem discovery"
  delay    = "1h"
  lifetime = "30d"

  condition {
    macro = "{#FSTYPE}"
    value = "^(ext|xfs|btrfs)"
  }
}

resource "zabbix_proto_item_agent" "fs_size" {
  hostid    = zabbix_template.app.id
  ruleid    = zabbix_lld_agent.fs.id
  key       = "vfs.fs.size[{#FSNAME},total]"
  name      = "Filesystem {#FSNAME} total size"
  valuetype = "unsigned"
  delay     = "5m"
}

# ----------------------------------------------------------------- host

resource "zabbix_host" "web01" {
  host      = "web01.example.com"
  name      = "Web server 01"
  groups    = [zabbix_hostgroup.linux.id]
  templates = [zabbix_template.app.id]

  interface {
    type = "agent"
    ip   = "192.168.1.10"
    port = 10050
    main = true
  }

  inventory_mode = "manual"
  inventory {
    location = "DC1 rack 12"
  }

  macro {
    name  = "{$APP_PORT}"
    value = "9090"   # override template default
  }

  tag {
    key   = "env"
    value = "production"
  }
}

# ------------------------------------------------- alerting (webhook mediatype)

resource "zabbix_mediatype" "slack" {
  name    = "Slack"
  type    = "webhook"
  # replace with your actual webhook script
  script  = <<-JS
    var params = JSON.parse(value);
    var req = new HttpRequest();
    req.addHeader("Content-Type: application/json");
    req.post(params.url, JSON.stringify({text: params.subject + "\n" + params.message}));
    return "OK";
  JS
  timeout = "10s"
  status  = "enabled"

  parameters {
    name  = "url"
    value = var.slack_webhook_url
  }
}

resource "zabbix_action" "notify_ops" {
  name        = "Notify ops on problem"
  eventsource = "trigger"
  esc_period  = "30m"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "0"  # host group
      operator      = "0"  # equals
      value         = zabbix_hostgroup.linux.id
    }
  }

  operations {
    operationtype = "0"  # send message
    esc_step_from = "1"
    esc_step_to   = "1"
    opmessage {
      default_msg  = "1"
      mediatypeid  = zabbix_mediatype.slack.id
    }
    opmessage_grp {
      usrgrpid = data.zabbix_usergroup.ops.id
    }
  }
}

data "zabbix_usergroup" "ops" {
  name = "Zabbix administrators"
}

# ------------------------------------------- maintenance window (every Sunday)

resource "zabbix_maintenance" "weekly_patching" {
  name             = "Weekly patching window"
  maintenance_type = "with_data"
  active_since     = "1800000000"
  active_till      = "1900000000"
  groups           = [zabbix_hostgroup.linux.id]

  tag {
    key      = "env"
    operator = "0"
    value    = "production"
  }

  timeperiod {
    type       = "weekly"
    every      = "1"
    dayofweek  = "64"   # Sunday
    start_time = "7200" # 02:00
    period     = "7200" # 2h
  }
}

# ---------------------------------------------------------- access management

resource "zabbix_role" "monitoring" {
  name       = "monitoring-readonly"
  type       = "user"
  api_access = false
}

resource "zabbix_usergroup" "team" {
  name         = "Platform team"
  users_status = true

  hostgroup_rights {
    id         = zabbix_hostgroup.linux.id
    permission = "read-only"
  }
}

resource "zabbix_user" "alice" {
  username = "alice"
  name     = "Alice"
  surname  = "Smith"
  passwd   = var.alice_password
  roleid   = zabbix_role.monitoring.id
  usrgrps  = [zabbix_usergroup.team.id]
  lang     = "en_US"
  timezone = "UTC"
}

# --------------------------------------------------- MFA-safe automation token

resource "zabbix_token" "ci" {
  name        = "ci-automation"
  description = "Used by CI to push metrics"
  userid      = zabbix_user.alice.id
  enabled     = true
}

# use the secret in outputs or pass it to a secret store
output "ci_token" {
  value     = zabbix_token.ci.token
  sensitive = true
}

# ------------------------------------------------------------------ variables

variable "slack_webhook_url" {
  type      = string
  sensitive = true
}

variable "alice_password" {
  type      = string
  sensitive = true
}
