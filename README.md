# Overview

A [Terraform](terraform.io) provider for [Zabbix](https://www.zabbix.com)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

<img src="https://assets.zabbix.com/img/logo/zabbix_logo_500x131.png" width="500px">

# Index

## Data Sources

* [zabbix_host](#datazabbix_host)
* [zabbix_hostgroup](#datazabbix_hostgroup)
* [zabbix_template](#datazabbix_template)
* [zabbix_application](#datazabbix_application)
* [zabbix_proxy](#datazabbix_proxy)

## Resources

* [zabbix_host](#zabbix_host)
* [zabbix_hostgroup](#zabbix_hostgroup)
* [zabbix_template](#zabbix_template)
* [zabbix_application](#zabbix_application)
* [zabbix_graph / zabbix_proto_graph](#zabbix_graph--zabbix_proto_graph)
* [zabbix_trigger / zabbix_proto_trigger](#zabbix_trigger--zabbix_proto_trigger)
* [zabbix_item_agent / zabbix_proto_item_agent](#zabbix_item_agent--zabbix_proto_item_agent)
* [zabbix_item_snmp / zabbix_proto_item_snmp](#zabbix_item_snmp--zabbix_proto_item_snmp)
* [zabbix_item_simple / zabbix_proto_item_simple](#zabbix_item_simple--zabbix_proto_item_simple)
* [zabbix_item_http / zabbix_proto_item_http](#zabbix_item_http--zabbix_proto_item_http)
* [zabbix_item_trapper / zabbix_proto_item_trapper](#zabbix_item_trapper--zabbix_proto_item_trapper)
* [zabbix_item_aggregate / zabbix_proto_item_aggregate](#zabbix_item_aggregate--zabbix_proto_item_aggregate)
* [zabbix_item_external / zabbix_proto_item_external](#zabbix_item_external--zabbix_proto_item_external)
* [zabbix_item_internal / zabbix_proto_item_internal](#zabbix_item_internal--zabbix_proto_item_internal)
* [zabbix_item_dependent / zabbix_proto_item_dependent](#zabbix_item_dependent--zabbix_proto_item_dependent)
* [zabbix_item_calculated / zabbix_proto_item_calculated](#zabbix_item_calculated--zabbix_proto_item_calculated)
* [zabbix_item_snmptrap / zabbix_proto_item_snmptrap](#zabbix_item_snmptrap--zabbix_proto_item_snmptrap)
* [zabbix_lld_agent](#zabbix_lld_agent)
* [zabbix_lld_trapper](#zabbix_lld_trapper)
* [zabbix_lld_simple](#zabbix_lld_simple)
* [zabbix_lld_external](#zabbix_lld_external)
* [zabbix_lld_internal](#zabbix_lld_internal)
* [zabbix_lld_dependent](#zabbix_lld_dependent)
* [zabbix_lld_snmp](#zabbix_lld_snmp)
* [zabbix_lld_http](#zabbix_lld_http)

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
  username = "<api_user>"                         # or use environment variable `ZABBIX_USER`
  password = "<api_password>"                     # or use environment variable `ZABBIX_PASS`
  url = "http://example.com/api_jsonrpc.php"      # or use environment variable `ZABBIX_URL`
  
  # Optional

  # Disable TLS verfication (false by default)
  tls_insecure = true

  # Serialize Zabbix API calls (false by default)
  # Note: race conditions have been observed, enable this if required
  serialize = true
}
```

## Data Sources

### data.zabbix_host
[index](#index)

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

### data.zabbix_hostgroup
[index](#index)

```hcl
data "zabbix_hostgroup" "example" {
  name = "Friendly Name"
}
```

#### Argument Reference

* name - (Required) Displayname of hostgroup

#### Attributes Reference

* name - Displayname of hostgroup

### data.zabbix_template
[index](#index)

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

### data.zabbix_application

```hcl
data "zabbix_template" "example" {
  name = "Friendly Name"
  hostid = "1245"
}
```

#### Argument Reference

* name - (Required) Name of template
* hostid - (Optional) ID of host / template

#### Attributes Reference

* name - Name of Template
* hostid - ID of host / template

### data.zabbix_proxy
[index](#index)

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
[index](#index)

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

    # if zabbix version >= 5 and type is snmp
    snmp_version = "3"
    snmp_community = "public"
    snmp3_authpassphrase = "supersecretpassword"
    snmp3_authprotocol = "md5"
    snmp3_contextname = "context"
    snmp3_privpassphrase = "anotherpassword"
    snmp3_privprotocol = "des"
    snmp3_securitylevel = "noauthnopriv"
    snmp3_securityname = "secname"
  }

  macro {
    key = "{$MACROABC}"
    value = "test_value_one"
  }

  inventory_mode = "manual"
  inventory {
    alias = "bob"
    notes = "test note"
  }
}
```

#### Argument Reference

* host - (Required) FQDN of host
* name - (Optional) Displayname of host
* groups - (Required) List of hostgroup IDs
* templates - (Optional) List of template IDs
* proxyid - (Optional) Zabbix proxy id for this host
* macro - (Optional) List of Macros
    * macro.#.name - Macro name
    * macro.#.value - Macro value
* interface - (Required) Host Interfaces
    * interface.#.type - (Required) Type of interface (agent,snmp,ipmi,jmx)
    * interface.#.dns - (Optional) DNS name
    * interface.#.ip - (Optional) IP Address
    * interface.#.main - (Optional) Primary interface of this type
    * interface.#.port - (Optional) Interface port to use
* inventory_mode - (Optional) Defaults to "disabled", can be one of "disabled", "manual" or "automatic"
* inventory - (Optional) Requires inventory_mode be set to one of "manual" or "automatic".
  Block contains key/value pairs as supported by your zabbix inventory version https://www.zabbix.com/documentation/5.0/manual/api/reference/host/object#host

The following only have affect on zabbix versions >= 5 and where type == snmp

* interface.#.snmp_version - (Optional) SNMP Version, defaults to 2, one of (1, 2, 3)
* interface.#.snmp_community - (Optional) SNMPv1/v2 community string, defaults to {$SNMP_COMMUNITY}
* interface.#.snmp3_authpassphrase - (Optional) SNMPv3 Auth passphrase, defaults to {$SNMP3_AUTHPASSPHRASE}
* interface.#.snmp3_authprotocol - (Optional) SNMPv3 Auth protocol, defaults to sha, one of (md5, sha)
* interface.#.snmp3_contextname - (Optional) SNMPv3 Context Name, defaults to {$SNMP3_CONTEXTNAME} 
* interface.#.snmp3_privpassphrase - (Optional) SNMPv3 Priv passphrase, defaults to {$SNMP3_PRIVPASSPHRASE}
* interface.#.snmp3_privprotocol - (Optional) SNMPv3 Priv protocol, defaults to aes, one of (des, aes)
* interface.#.snmp3_securitylevel - (Optional) SNMPv3 Security Level, defaults to authpriv, one of (noauthnopriv, authnopriv, authpriv)
* interface.#.snmp3_securityname - (Optional) SNMPv3 Security Name, defaults to {$SNMP3_SECURITYNAME}

#### Attributes Reference

Same as arguments, plus:

* interface.#.id - Generated Interface ID
* macro.#.id - Generated macro ID


### zabbix_hostgroup
[index](#index)

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
[index](#index)

```hcl
resource "zabbix_template" "example" {
  host = "template internal name"
  name = "Friendly Name"

  groups = [ "1234" ]
  description = "Template Description"

  templates = [ "5678" ]
  
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
* templates - (Optional) List of template IDs to link to this template
* macro - (Optional) List of Macros
    * macro.#.name - Macro name
    * macro.#.value - Macro value

#### Attributes Reference

Same as arguments, plus:

* macro.#.id - Generated macro ID

### zabbix_application
[index](#index)

```hcl
resource "zabbix_application" "example" {
  name = "Application Name"
  hostid = "1234"
}
```

#### Argument Reference

* name - (Required) Name of application
* hostid - (Required) ID of host / template

#### Attributes Reference

Same as arguments

### zabbix_graph / zabbix_proto_graph
[index](#index)

```hcl
resource "zabbix_graph" "example" {
  name = "Graph Name"
  height = "100"
  width = "100"
  type = "normal"
  percent_left = "0"
  percent_right = "0"

  do3d = true
  legend = true
  work_period = true

  ymax = "100"
  ymax_itemid = "1234"
  ymax_type = "calculated"
  
  ymin = "100"
  ymin_itemid = "1234"
  ymin_type = "calculated"

  item {
    color = "#ffffff"
    itemid = "1234"
    function = "min"
    drawtype = "line"
    sortorder = "0"
    type = "simple"
    yaxis_side = "left"
  }
}
```

#### Argument Reference

* name - (Required) Name of graph
* height - (Required) Height of graph
* width - (Required) Width of graph
* type - (Optional) Graph type, defaults to "normal" one of "normal", "stacked", "pie", "exploded"
* percent_left - (Optional) Left percentile, defaults to 0
* percent_right - (Optional) Right percentile, defaults to 0
* do3d - (Optional) 3D graph, defaults to false
* legend - (Optional) Show legend, defaults to true
* work_period - (Optional) Show work period, defaults to true
* ymax - (Optional) Max value of y axis, defaults to 100
* ymax_itemid - (Optional) ItemID to use as the y axis maximum
* ymax_type - (Optional) Type of yaxis max limit, defaults to "calculated", one of "calculated", "fixed", "item"
* ymin - (Optional) Min value of y axis, defaults to 0
* ymin_itemid - (Optional) ItemID to use as the y axis minimum
* ymin_type - (Optional) Type of yaxis min limit, defaults to "calculated", one of "calculated", "fixed", "item"
* item - (Required) List of item objects
    * color - (Required) Item Color
    * itemid - (Required) ID of item
    * function - (Optional) Data Function, defaults to "min", one of "min", "average", "max", "all", "last"
    * drawtype - (Optional) Draw Type, defaults to "line", one of "line", "filled", "bold", "dot", "dashed", "gradient"
    * sortorder - (Optional) Position of item in graph, defaults to 0
    * type - (Optional) Type of graph item, defaults to "simple", one of "simple", "sum"
    * yaxis_side - (Optional) Side of Y Axis, defaults to "left", one of "left", "right"

#### Attributes Reference

Same as arguments

### zabbix_trigger / zabbix_proto_trigger
[index](#index)

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
[index](#index)

```hcl
resource "zabbix_item_agent" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"
  valuetype = "unsigned"

  delay = "1m"
  history = "90d"
  trends = "365d"

  # only for proto_item
  ruleid = "8989"
  applications = [ "4567" ]

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* active - (Optional) zabbix active agent (defaults to false)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_snmp / zabbix_proto_item_snmp
[index](#index)

```hcl
j
resource "zabbix_item_snmp" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"
  valuetype = "unsigned"
  
  # only for proto_item
  ruleid = "8989"

  applications = [ "4567" ]

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  snmp_oid = "1.2.3.4
  
  # below should only be used on zabbix versions < 5
  snmp_version = "3"
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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate
* snmp_oid - (Required) SNMP OID Number

The following only have an effect in zabbix versions < 5

* snmp_version - (Optional) SNMP Version, defaults to 2, one of (1, 2, 3)
* snmp_community - (Optional) SNMPv1/v2 community string, defaults to {$SNMP_COMMUNITY}
* snmp3_authpassphrase - (Optional) SNMPv3 Auth passphrase, defaults to {$SNMP3_AUTHPASSPHRASE}
* snmp3_authprotocol - (Optional) SNMPv3 Auth protocol, defaults to sha, one of (md5, sha)
* snmp3_contextname - (Optional) SNMPv3 Context Name, defaults to {$SNMP3_CONTEXTNAME} 
* snmp3_privpassphrase - (Optional) SNMPv3 Priv passphrase, defaults to {$SNMP3_PRIVPASSPHRASE}
* snmp3_privprotocol - (Optional) SNMPv3 Priv protocol, defaults to aes, one of (des, aes)
* snmp3_securitylevel - (Optional) SNMPv3 Security Level, defaults to authpriv, one of (noauthnopriv, authnopriv, authpriv)
* snmp3_securityname - (Optional) SNMPv3 Security Name, defaults to {$SNMP3_SECURITYNAME}

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_simple / zabbix_proto_item_simple
[index](#index)

```hcl
resource "zabbix_item_simple" "example" {
  hostid = "1234"
  key = "net.tcp.service[ftp,,155]"
  name = "Item Name"
  valuetype = "unsigned"

  # only for proto_item
  ruleid = "8989"

  applications = [ "4567" ]

  delay = "1m"
  history = "90d"
  trends = "365d"

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number


### zabbix_item_http / zabbix_proto_item_http
[index](#index)

```hcl
resource "zabbix_item_http" "example" {
  hostid = "1234"
  key = "http_value_search"
  name = "Item Name"
  valuetype = "unsigned"

  # only for proto_item
  ruleid = "8989"

  applications = [ "4567" ]

  delay = "1m"
  history = "90d"
  trends = "365d"

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

  auth_type = "basic"
  username = "bob"
  password = "supersecretpassword"

  headers = {
    "Accept": "application/json"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
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
* applications - (Optional) list of application IDs to associate
* auth_type - (Optional) Authentication type, defaults to "none", one of none, basic, digest, ntlm, kerberos
* username - (Optional) Username
* password - (Optional) Password
* headers - (Optional) Map of http headers to include

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_trapper / zabbix_proto_item_trapper
[index](#index)

```hcl
resource "zabbix_item_trapper" "example" {
  hostid = "1234"
  key = "trapper_item_key"
  name = "Item Name"
  valuetype = "unsigned"

  # only for proto_item
  ruleid = "8989"

  applications = [ "4567" ]

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_aggregate / zabbix_proto_item_aggregate
[index](#index)

```hcl
resource "zabbix_item_aggregate" "example" {
  hostid = "1234"
  key = "grpsum()"
  name = "Item Name"
  valuetype = "unsigned"

  delay = "1m"
  history = "90d"
  trends = "365d"

  # only for proto_item
  ruleid = "8989"

  applications = [ "4567" ]

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_external / zabbix_proto_item_external
[index](#index)

```hcl
resource "zabbix_item_external" "example" {
  hostid = "1234"
  key = "script[\"argv1\",\"argv2\"]"
  name = "Item Name"
  interfaceid = "5678"
  valuetype = "unsigned"
  delay = "1m"
  history = "90d"
  trends = "365d"

  # only for proto_item
  ruleid = "8989"
  
  applications = [ "4567" ]
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach item to
* key - (Required) Item Key
* name - (Required) Item Name
* interfaceid - (Required) Host interface ID
* valuetype - (Required) Item valuetype, one of: (float, character, log, unsigned, text)
* delay - (Optional) Item collection interval, defaults to 1m
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_internal / zabbix_proto_item_internal
[index](#index)

```hcl
resource "zabbix_item_internal" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"
  valuetype = "unsigned"

  delay = "1m"
  history = "90d"
  trends = "365d"

  # only for proto_item
  ruleid = "8989"
  
  applications = [ "4567" ]

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_dependent / zabbix_proto_item_dependent
[index](#index)

```hcl
resource "zabbix_item_dependent" "example" {
  hostid = "1234"
  key = "custom.hostname"
  name = "Item Name"
  valuetype = "text"

  master_itemid = "12344"

  # only for proto_item
  ruleid = "8989"
  
  applications = [ "4567" ]

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_calculated / zabbix_proto_item_calculated
[index](#index)

```hcl
resource "zabbix_item_dependent" "example" {
  hostid = "1234"
  key = "custom.hostname"
  name = "Item Name"
  valuetype = "text"

  formula = "1+1"

  # only for proto_item
  ruleid = "8989"
  
  applications = [ "4567" ]

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
* formula - (Required) Calculated Item Formula
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_item_snmptrap / zabbix_proto_item_snmptrap
[index](#index)

```hcl
resource "zabbix_item_snmptrap" "example" {
  hostid = "1234"
  key = "custom.hostname"
  name = "Item Name"
  valuetype = "text"

  # only for proto_item
  ruleid = "8989"
  
  applications = [ "4567" ]

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
* history - (Optional) Item retention period
* trends - (Optional) Item trend period
* preprocessor - (Optional) Item Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* ruleid - (Required for proto_item) LLD Discovery rule ID to attach prototype item to
* applications - (Optional) list of application IDs to associate

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_agent
[index](#index)

```hcl
resource "zabbix_lld_agent" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"

  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }

  active = true
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* active - (Optional) zabbix active agent (defaults to false)
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_trapper
[index](#index)

```hcl
resource "zabbix_lld_trapper" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_simple
[index](#index)

```hcl
resource "zabbix_lld_simple" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"
  
  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_external
[index](#index)

```hcl
resource "zabbix_lld_external" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"
  
  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_internal
[index](#index)

```hcl
resource "zabbix_lld_internal" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"
  
  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_dependent
[index](#index)

```hcl
resource "zabbix_lld_dependent" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"
  
  master_itemid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* master_itemid - (Required) ItemID this depends on

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_snmp
[index](#index)

```hcl
resource "zabbix_lld_snmp" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"
  
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
  
  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
  }
}
```

#### Argument Reference

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
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

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number

### zabbix_lld_http
[index](#index)

```hcl
resource "zabbix_lld_http" "example" {
  hostid = "1234"
  key = "zabbix.hostname"
  name = "Item Name"

  delay = "1m"
  lifetime = "1d"
  evaltype = "and"
  
  interfaceid = "5678"

  preprocessor {
    type = "5"
    params = ["param a", "param b"]
    error_handler = "1"
    error_handler_params = ""
  }

  condition {
    macro = "{#name}"
    value = "^blah"
    operator = "match"
  }

  macro {
    macro = "{#name}"
    path = "$.bob"
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

* hostid - (Required) Host/Template ID to attach LLD Rule to
* key - (Required) LLD Key
* name - (Required) LLD Name
* delay - (Optional) LLD collection interval, defaults to 1m
* lifetime - (Optional) Discovery Item lifetime, defaults to 30d
* evaltype - (Optional) Discovery Filter Evaluation type, defaults to andor
* formula - (Optional) Filter formula
* preprocessor - (Optional) LLD Preprocessors
    * type - (Required) Preprocessor type [docs](https://www.zabbix.com/documentation/current/manual/api/reference/item/object)
    * params - (Optional) Preprocessor params
    * error_handler - (Optional) error handler type (see above docs, only relevent in > 4.0)
    * error_handler_params - (Optional) error handler params (see above docs, only relevent in > 4.0)
* condition - (Optional) LLD Filters
    * macro - (Required) Filter macro name
    * value - (Required) Filter Regex
    * operator - (Optional) Filter operator, defaults to "match"
* macro - (Optional) LLD Macros
    * macro - (Required) Macro name
    * path - (Required) Macro JSON path
* interfaceid - (Optional) Host interface ID, defaults to 0 (not required for template attachment)
* url - (Required) URL to fetch
* request_method - (Optional) Method to use, defaults to "get", one of (get, post, put, head)
* post_type - (Optional) Post type to use, defaults to "body", one of (body, headers, both)
* status_codes - (Optional) Status codes to detect, defaults to 200
* timeout - (Optional) Request timeout, defaults to 3s
* verify_host (Optional) TLS host verification, defaults to true
* verify_peer (Optional) TLS peer verification, defaults to true

#### Attributes Reference

Same as arguments, plus:

* preprocessor.#.id - Preprocessor assigned ID number
