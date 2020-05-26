# Overview

A [Terraform](terraform.io) provider for [Zabbix](https://www.zabbix.com)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

<img src="https://assets.zabbix.com/img/logo/zabbix_logo_500x131.png" width="500px">

# Index

[zabbix_host](#zabbix_host)
[zabbix_hostgroup](#zabbix_hostgroup)
[zabbix_template](#zabbix_template)
[zabbix_proxy](#zabbix_proxy)
[zabbix_host](#zabbix_host)
[zabbix_hostgroup](#zabbix_hostgroup)
[zabbix_template](#zabbix_template)
[zabbix_trigger](#zabbix_trigger)
[zabbix_item_agent / zabbix_proto_item_agent](#zabbix_item_agent-/-zabbix_proto_item_agent)
[zabbix_item_snmp / zabbix_proto_item_snmp](#zabbix_item_snmp-/-zabbix_proto_item_snmp)
[zabbix_item_simple / zabbix_proto_item_simple](#zabbix_item_simple-/-zabbix_proto_item_simple)
[zabbix_item_http / zabbix_proto_item_http](#zabbix_item_http-/-zabbix_proto_item_http)
[zabbix_item_trapper / zabbix_proto_item_trapper](#zabbix_item_trapper-/-zabbix_proto_item_trapper)
[zabbix_item_aggregate / zabbix_proto_item_aggregate](#zabbix_item_aggregate-/-zabbix_proto_item_aggregate)
[zabbix_item_external / zabbix_proto_item_external](#zabbix_item_external-/-zabbix_proto_item_external)
[zabbix_item_internal / zabbix_proto_item_internal](#zabbix_item_internal-/-zabbix_proto_item_internal)
[zabbix_item_dependent / zabbix_proto_item_dependent](#zabbix_item_dependent-/-zabbix_proto_item_dependent)

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

All resources support terraform resource importing using zabbix ID numbers

# Templates to Terraform

The script `utils/template2terraform` provides the capabilities to convert (some of) a Zabbix XML template into Terraform HCL.

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
* proxyid - Proxy ID
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

### zabbix_proxy

```hcl
data "zabbix_proxy" "example" {
  host = "proxy.name"
}
```

#### Argument Reference

* host - (Required) Name of proxy

#### Attributes Reference

* host - name of proxy

## Resources

### zabbix_host

```hcl
resource "zabbix_host" "example" {
  host = "server.example.com"
  name = "Friendly Name"

  enabled = false

  groups = [ "1234" ]
  templates = [ "5678" ]
  proxyid = "7890"

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
* proxyid - (Optional) Zabbix proxy id for this host
* macro - (Optional) List of Macros
    * macro.#.name - Macro name
    * macro.#.value - Macro value

#### Attributes Reference

Same as arguments, plus:

* interface.#.id - Generated Interface ID
* macro.#.id - Generated macro ID


### zabbix_hostgroup

```hcl
resource "zabbix_hostgroup" "example" {
  name = "Friendly Name"
}
```

#### Argument Reference

* name - (Required) Displayname of hostgroup

#### Attributes Reference

Same as arguments

### zabbix_template

```hcl
resource "zabbix_template" "example" {
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
resource "zabbix_trigger" "example" {
  name = "Trigger Name"
  expression = "{trigger:expression.last()} > 10"
  comments = "Trigger Comments"

  priority = "high"
  enabled = false

  multiple = false
  url = "http://example.com/triggerdocs"
  recovery_none = false
  recovery_expression = "{trigger:expression.last()} > 15"

  correlation_tag = "example"
  manual_close = false

  dependencies = [ "1234" ]

  tag {
    key = "service_type"
    value = "webserver"
  }
}
```

#### Note

When referencing hosts, templates or items within the expression, or recovery_expression, ensure you reference other resources via an attribute lookup.

Without this, simply specifying the raw strings, will prevent terraform from correctly understanding the dependencies between triggers and other resources.

Example
```
# Bad
expression = "{Template Name:itemname.last()}>0"

# Good
expression = "{${zabbix_template.a.name}:${zabbix_item_snmp.b.key}.last()}>0"
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
* tag - (Optional) List of Tags
    * tag.#.key - (Required) Tag Key
    * tag.#.value - (Optional) Tag Value (for tags with a name and value)

#### Attributes Reference

Same as arguments

### zabbix_item_agent / zabbix_proto_item_agent

```hcl
resource "zabbix_item_agent" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"
  valuetype = "unsigned"

  delay = "1m"

  # only for proto_item
  ruleid = "8989"

  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  active = true
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* active - (Optional) zabbix active agent (defaults to false)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_snmp / zabbix_proto_item_snmp

```hcl
j
resource "zabbix_item_snmp" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"
  valuetype = "unsigned"
  
  # only for proto_item
  ruleid = "8989"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  snmp_version = "3"
  snmp_oid = "1.2.3.4
  
  snmp_community = "public"

  snmp3_authpassphrase = "supersecretpassword"
  snmp3_authprotocol = "md5"
  snmp3_contextname = "context"
  snmp3_privpassphrase = "anotherpassword"
  snmp3_privprotocol = "des"
  snmp3_securitylevel = "noauthnopriv"
  snmp3_securityname = "secname"
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* snmp_version - (Optional) SNMP Version, defaults to 2, one of (1, 2, 3)
* snmp_oid - (Required) SNMP OID Number
* snmp_community - (Optional) SNMPv1/v2 community string, defaults to {$SNMP_COMMUNITY}
* snmp3_authpassphrase - (Optional) SNMPv3 Auth passphrase, defaults to {$SNMP3_AUTHPASSPHRASE}
* snmp3_authprotocol - (Optional) SNMPv3 Auth protocol, defaults to sha, one of (md5, sha)
* snmp3_contextname - (Optional) SNMPv3 Context Name, defaults to {$SNMP3_CONTEXTNAME} 
* snmp3_privpassphrase - (Optional) SNMPv3 Priv passphrase, defaults to {$SNMP3_PRIVPASSPHRASE}
* snmp3_privprotocol - (Optional) SNMPv3 Priv protocol, defaults to aes, one of (des, aes)
* snmp3_securitylevel - (Optional) SNMPv3 Security Level, defaults to authpriv, one of (noauthnopriv, authnopriv, authpriv)
* snmp3_securityname - (Optional) SNMPv3 Security Name, defaults to {$SNMP3_SECURITYNAME}
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_simple / zabbix_proto_item_simple

```hcl
resource "zabbix_item_simple" "example" {
  hostid = "1234"
  key = "net.tcp.service[ftp,,155]"
  name = "Item Name"
  valuetype = "unsigned"

  # only for proto_item
  ruleid = "8989"

  delay = "1m"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number


### zabbix_item_http / zabbix_proto_item_http

```hcl
resource "zabbix_item_http" "example" {
  hostid = "1234"
  key = "http_value_search"
  name = "Item Name"
  valuetype = "unsigned"

  # only for proto_item
  ruleid = "8989"

  delay = "1m"

  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  url = "http://example.com"
  request_method = "post"
  post_type = "body"
  posts = "{}"
  status_codes = "200"
  timeout = "3s"
  verify_host = true
  verify_peer = true
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)

* url - (Required) URL to fetch
* request_method - (Optional) Method to use, defaults to "get", one of (get, post, put, head)
* post_type - (Optional) Post type to use, defaults to "body", one of (body, headers, both)
* status_codes - (Optional) Status codes to detect, defaults to 200
* timeout - (Optional) Request timeout, defaults to 3s
* verify_host (Optional) TLS host verification, defaults to true
* verify_peer (Optional) TLS peer verification, defaults to true
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_trapper / zabbix_proto_item_trapper

```hcl
resource "zabbix_item_trapper" "example" {
  hostid = "1234"
  key = "trapper_item_key"
  name = "Item Name"
  valuetype = "unsigned"

  # only for proto_item
  ruleid = "8989"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_aggregate / zabbix_proto_item_aggregate

```hcl
resource "zabbix_item_aggregate" "example" {
  hostid = "1234"
  key = "grpsum()"
  name = "Item Name"
  valuetype = "unsigned"

  delay = "1m"

  # only for proto_item
  ruleid = "8989"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_external / zabbix_proto_item_external

```hcl
resource "zabbix_item_external" "example" {
  hostid = "1234"
  key = "script[\"argv1\",\"argv2\"]"
  name = "Item Name"
  interfaceid = "5678"
  valuetype = "unsigned"
  delay = "1m"

  # only for proto_item
  ruleid = "8989"
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* interfaceid - (Required) Host interface ID
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_internal / zabbix_proto_item_internal

```hcl
resource "zabbix_item_internal" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"
  valuetype = "unsigned"

  delay = "1m"

  # only for proto_item
  ruleid = "8989"

  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_dependent / zabbix_proto_item_dependent

```hcl
resource "zabbix_item_dependent" "example" {
  hostid = "1234"
  key = "custom.hostname"
  name = "Item Name"
  valuetype = "text"

  master_itemid = "12344"

  # only for proto_item
  ruleid = "8989"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* master_itemid - (Required) Master Item ID
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number