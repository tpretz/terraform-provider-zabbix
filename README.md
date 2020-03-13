# Overview

A [Terraform](terraform.io) provider for [Zabbix](https://www.zabbix.com)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

<img src="https://assets.zabbix.com/img/logo/zabbix_logo_500x131.png" width="500px">

# Requirements

- Access to Zabbix API over http or https


# Using the provider

Build or download, and install the appropriate binary into your terraform plugins directory.

[Plugin Basics](https://www.terraform.io/docs/plugins/basics.html#installing-plugins)

# Status

This integration is not feature complete and covers a limited set of Zabbix features.

# Testing

No Testing has yet been added to this repository

# Usage

## Provider

Instantiate an instance of the provider.

```
provider "zabbix" {
  # Required
  username = "<api_user>"
  password = "<api_password>"
  url = "http://example.com/api_jsonrpc.php"
  
  # Optional

  # Disable TLS verfication (false by default)
  tls_insecure = true

  # Serialize Zabbix API calls (false by default)
  # Note: race conditions have been observed, enable this if required
  serialize = true
}
```

## Data Sources

### zabbix_host

```hcl
data "zabbix_host" "example" {
  host = "server.example.com"
  name = "Friendly Name"
  hostid = "1234"
}
```

#### Argument Reference

* host - (Optional) FQDN of host
* name - (Optional) Displayname of host
* hostid - (Optional) Zabbix host UUID

#### Attributes Reference

* host - FQDN of host
* name - Displayname of host
* enabled - Host enabled for monitoring
* interface - Host Interfaces
    * interface.#.id - Generated Interface ID
    * interface.#.dns - DNS name
    * interface.#.ip - IP Address
    * interface.#.main - Primary interface of this type
    * interface.#.port - Interface port to use
    * interface.#.type - Type of interface (agent,snmp,ipmi,jmx)
* groups - List of hostgroup IDs
* templates - List of template IDs
* macro - List of Macros
    * macro.#.id - Generated macro ID
    * macro.#.name - Macro name
    * macro.#.value - Macro value

### zabbix_hostgroup

```hcl
data "zabbix_hostgroup" "example" {
  name = "Friendly Name"
}
```

#### Argument Reference

* name - (Required) Displayname of hostgroup

#### Attributes Reference

* name - Displayname of hostgroup

### zabbix_template

```hcl
data "zabbix_template" "example" {
  host = "template internal name"
  name = "Friendly Name"
}
```

#### Argument Reference

* host - (Optional) Name of Template
* name - (Optional) Displayname of template

#### Attributes Reference

* host - Name of Template
* name - Displayname of template
* description - description
* groups - List of hostgroup IDs
* macro - List of Macros
    * macro.#.id - Generated macro ID
    * macro.#.name - Macro name
    * macro.#.value - Macro value

## Resources

### zabbix_host

```hcl
data "zabbix_host" "example" {
  host = "server.example.com"
  name = "Friendly Name"

  enabled = false

  groups = [ "1234" ]
  templates = [ "5678" ]

  interface {
    type = "snmp"
    dns = "interface.dns.name"
    ip = "interface.ip.addr"

    main = false
    port = 1161
  }

  macro {
    key = "{$MACROABC}"
    value = "test_value_one"
  }
}
```

#### Argument Reference

* host - (Required) FQDN of host
* name - (Optional) Displayname of host
* interface - (Required) Host Interfaces
    * interface.#.type - (Required) Type of interface (agent,snmp,ipmi,jmx)
    * interface.#.dns - (Optional) DNS name
    * interface.#.ip - (Optional) IP Address
    * interface.#.main - (Optional) Primary interface of this type
    * interface.#.port - (Optional) Interface port to use
* groups - (Required) List of hostgroup IDs
* templates - (Optional) List of template IDs
* macro - (Optional) List of Macros
    * macro.#.name - Macro name
    * macro.#.value - Macro value

#### Attributes Reference

Same as arguments, plus:

* interface.#.id - Generated Interface ID
* macro.#.id - Generated macro ID


### zabbix_hostgroup

```hcl
data "zabbix_hostgroup" "example" {
  name = "Friendly Name"
}
```

#### Argument Reference

* name - (Required) Displayname of hostgroup

#### Attributes Reference

Same as arguments

### zabbix_template

```hcl
data "zabbix_template" "example" {
  host = "template internal name"
  name = "Friendly Name"

  groups = [ "1234" ]
  description = "Template Description"
  
  macro {
    key = "{$MACROABC}"
    value = "test_value_one"
  }
}
```

#### Argument Reference

* host - (Required) Name of Template
* name - (Optional) Displayname of template
* description - (Optional) Template description
* groups - (Required) List of hostgroup IDs
* macro - (Optional) List of Macros
    * macro.#.name - Macro name
    * macro.#.value - Macro value

#### Attributes Reference

Same as arguments, plus:

* macro.#.id - Generated macro ID

### zabbix_trigger

```hcl
data "zabbix_trigger" "example" {
  name = "Trigger Name"
  expression = "{trigger:expression.last()} > 10"
  comments = "Trigger Comments"

  priority = "high"
  enabled = false

  groups = [ "1234" ]
  description = "Template Description"
  multiple = false
  url = "http://example.com/triggerdocs"
  recovery_none = false
  recovery_expression = "{trigger:expression.last()} > 15"

  correlation_tag = "example"
  manual_close = false

  dependencies = [ "1234" ]
}
```

#### Argument Reference

* host - (Required) Trigger name
* expression - (Required) Trigger expression
* comments - (Optional) Trigger comments
* priority - (Optional) Trigger priority, defaults to non_classified, one of (not_classified, info, warn, average, high, disaster)
* enabled - (Optional) Enable trigger, defaults to true
* multiple - (Optional) Generate multiple alerts, defaults to false
* url - (Optional) Trigger URL
* recovery_none - (Optional) Disable recovery expressions, defaults to false
* recovery_expression - (Optional) Use this specific recovery expression
* correlation_tag - (Optional) Use this specific correlation tag
* manual_close - (Optional) Allow manual resolution
* dependencies - (Optional) List of Trigger IDs to be attached as dependencies

#### Attributes Reference

Same as arguments

### zabbix_item_agent
### zabbix_item_snmp
### zabbix_item_simple
### zabbix_item_http
### zabbix_item_trapper

