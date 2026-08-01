# zabbix_template_link (Resource)

Manages the contents (items, triggers, LLD rules) of a Zabbix template. This is a virtual
resource — it creates no object in Zabbix itself. Its purpose is to let Terraform
declaratively track which items and triggers belong to a template so they can be cleaned
up when removed from configuration.

When the `zabbix_template_link` resource is destroyed, the items and triggers in its state
are **not** deleted from Zabbix — Zabbix cascades deletions when the parent template is
removed.

## Example Usage

```hcl
resource "zabbix_hostgroup" "linux" {
  name = "Linux Servers"
}

resource "zabbix_template" "base" {
  host   = "base-template"
  groups = [zabbix_hostgroup.linux.id]
}

resource "zabbix_item_agent" "cpu" {
  hostid    = zabbix_template.base.id
  key       = "system.cpu.load[percpu,avg1]"
  name      = "CPU Load"
  valuetype = "float"
}

resource "zabbix_trigger" "cpu_high" {
  description = "CPU load high"
  expression  = "avg(/base-template/system.cpu.load[percpu,avg1],5m)>4"
  hostid      = zabbix_template.base.id
}

resource "zabbix_template_link" "base_contents" {
  template_id = zabbix_template.base.id

  item {
    item_id = zabbix_item_agent.cpu.id
  }

  trigger {
    trigger_id = zabbix_trigger.cpu_high.id
  }
}
```

## Argument Reference

| Name        | Type   | Required | Description                          |
|-------------|--------|----------|--------------------------------------|
| template_id | String | Yes      | ID of the template to manage         |
| item        | Set    | No       | Items belonging to this template     |
| trigger     | Set    | No       | Triggers belonging to this template  |
| lld_rule    | Set    | No       | LLD rules belonging to this template |

### item Block

| Name    | Type   | Required | Description |
|---------|--------|----------|-------------|
| item_id | String | Yes      | Item ID     |

### trigger Block

| Name       | Type   | Required | Description |
|------------|--------|----------|-------------|
| trigger_id | String | Yes      | Trigger ID  |

### lld_rule Block

| Name       | Type   | Required | Description |
|------------|--------|----------|-------------|
| lld_rule_id | String | Yes     | LLD rule ID |

## Import

Import by template ID:

```bash
terraform import zabbix_template_link.base_contents 12345
```
