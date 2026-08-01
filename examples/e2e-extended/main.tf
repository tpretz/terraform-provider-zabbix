// Extended end-to-end configuration covering the resources added for broader
// Zabbix 7.0 coverage: access management, alerting, operational objects,
// monitoring configuration and business services.
//
// The base monitoring objects (hosts, items, triggers, graphs, LLD) are covered
// by ../e2e/main.tf.

terraform {
  required_providers {
    zabbix = {
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}

provider "zabbix" {
  # ZABBIX_URL plus either ZABBIX_API_TOKEN, or ZABBIX_USER + ZABBIX_PASS
}

# ------------------------------------------------------------ access management

resource "zabbix_role" "automation" {
  name = "e2e-automation-role"
  # super_admin so the token below can manage global objects such as global
  # macros and regular expressions
  type = "super_admin"

  # API access is required for the token to be usable at all
  api_access = true
}

resource "zabbix_usergroup" "ops" {
  name         = "e2e-ops"
  gui_access   = "default"
  users_status = true

  hostgroup_rights {
    id         = zabbix_hostgroup.servers.id
    permission = "read-write"
  }
}

resource "zabbix_user" "operator" {
  username = "e2e-operator"
  name     = "E2E"
  surname  = "Operator"
  passwd   = "Tf-Pr0v1der-Str0ng!"
  roleid   = zabbix_role.automation.id
  usrgrps  = [zabbix_usergroup.ops.id]
  lang     = "en_US"
  timezone = "UTC"
}

resource "zabbix_token" "automation" {
  name        = "e2e-automation"
  description = "token used by automation, MFA safe"
  userid      = zabbix_user.operator.id
  enabled     = true
}

# --------------------------------------------------------------------- alerting

resource "zabbix_mediatype" "webhook" {
  name    = "e2e-webhook"
  type    = "webhook"
  script  = "return JSON.stringify({status: 'OK'});"
  timeout = "10s"
  status  = "enabled"

  parameters {
    name  = "url"
    value = "https://example.invalid/hook"
  }

  parameters {
    name  = "message"
    value = "{ALERT.MESSAGE}"
  }
}

resource "zabbix_script" "restart" {
  name         = "e2e-restart-service"
  type         = "script"
  scope        = "manual_host_action"
  command      = "systemctl restart myservice"
  execute_on   = "agent"
  description  = "restarts the service on the selected host"
  confirmation = "Restart the service?"
  host_access  = "write"
}

resource "zabbix_action" "notify" {
  name        = "e2e-notify-on-problem"
  eventsource = "trigger"
  status      = "enabled"
  esc_period  = "1h"

  filter {
    evaltype = "and_or"

    conditions {
      conditiontype = "0" # host group
      operator      = "0" # equals
      value         = zabbix_hostgroup.servers.id
    }
  }

  operations {
    operationtype = "0" # send message
    esc_step_from = "1"
    esc_step_to   = "1"

    opmessage {
      default_msg = "1"
      mediatypeid = zabbix_mediatype.webhook.id
    }

    opmessage_grp {
      usrgrpid = zabbix_usergroup.ops.id
    }
  }
}

# ------------------------------------------------------------------ operational

resource "zabbix_hostgroup" "servers" {
  name = "e2e-servers"
}

resource "zabbix_proxygroup" "edge" {
  name           = "e2e-edge"
  description    = "edge proxy group"
  failover_delay = "1m"
  min_online     = "1"
}

resource "zabbix_proxy" "edge_a" {
  name           = "e2e-edge-a"
  operating_mode = "active"
  description    = "active proxy in the edge group"
  proxy_groupid  = zabbix_proxygroup.edge.id
  local_address  = "127.0.0.1"
  local_port     = "10051"
}

resource "zabbix_maintenance" "weekly_patching" {
  name             = "e2e-weekly-patching"
  description      = "suppress alerts during the patch window"
  maintenance_type = "with_data"
  active_since     = "1800000000"
  active_till      = "1830000000"
  groups           = [zabbix_hostgroup.servers.id]

  tag {
    key      = "service"
    value    = "web"
    operator = "2"
  }

  timeperiod {
    type       = "weekly"
    every      = "1"
    dayofweek  = "64" # sunday
    start_time = "7200"
    period     = "10800"
  }
}

# ------------------------------------------------------ monitoring configuration

resource "zabbix_templategroup" "apps" {
  name = "e2e-apps"
}

resource "zabbix_template" "app" {
  host   = "e2e-ext-template"
  groups = [zabbix_templategroup.apps.id]
}

resource "zabbix_valuemap" "service_state" {
  hostid = zabbix_template.app.id
  name   = "e2e service state"

  mappings {
    type     = "equal"
    value    = "0"
    newvalue = "down"
  }

  mappings {
    type     = "equal"
    value    = "1"
    newvalue = "up"
  }

  mappings {
    type     = "default"
    newvalue = "unknown"
  }
}

resource "zabbix_global_macro" "environment" {
  macro       = "{$E2E_ENVIRONMENT}"
  value       = "acceptance"
  description = "environment marker used by the e2e config"
}

resource "zabbix_regexp" "log_errors" {
  name        = "e2e-log-errors"
  test_string = "ERROR: something failed"

  expressions {
    expression      = "^ERROR:"
    expression_type = "char_included"
    case_sensitive  = true
  }
}

# --------------------------------------------------------- business services

resource "zabbix_service" "web" {
  name      = "e2e-web-service"
  algorithm = "most_critical_child"
  sortorder = "0"

  problem_tags {
    tag      = "service"
    operator = "0"
    value    = "web"
  }

  tags {
    tag   = "tier"
    value = "frontend"
  }
}

resource "zabbix_sla" "web_availability" {
  name           = "e2e-web-sla"
  period         = "weekly"
  slo            = "99.9"
  effective_date = "1800000000"
  timezone       = "UTC"
  status         = "enabled"

  service_tags {
    tag      = "tier"
    operator = "0"
    value    = "frontend"
  }

  schedule {
    period_from = "0"
    period_to   = "86400"
  }
}

resource "zabbix_httptest" "homepage" {
  name    = "e2e-homepage-check"
  hostid  = zabbix_template.app.id
  delay   = "1m"
  retries = 2

  steps {
    name          = "fetch homepage"
    url           = "http://example.invalid/"
    no            = 1
    status_codes   = "200"
    required      = "Example"
    timeout       = "15s"
  }
}

# ---------------------------------------------------------------------- outputs

output "role_id" {
  value = zabbix_role.automation.id
}

output "user_id" {
  value = zabbix_user.operator.id
}

output "automation_token_created" {
  # the token itself is sensitive, so only assert that one was generated
  value     = zabbix_token.automation.token != ""
  sensitive = true
}

output "action_id" {
  value = zabbix_action.notify.id
}

output "proxy_in_group" {
  value = zabbix_proxy.edge_a.proxy_groupid == zabbix_proxygroup.edge.id
}

output "maintenance_id" {
  value = zabbix_maintenance.weekly_patching.id
}

output "valuemap_mapping_count" {
  value = length(zabbix_valuemap.service_state.mappings)
}

output "sla_id" {
  value = zabbix_sla.web_availability.id
}
