 # Overview

A [Terraform](terraform.io) provider for [Zabbix](https://www.zabbix.com). Based on [tpretz/terraform-provider-zabbix](https://github.com/tpretz/terraform-provider-zabbix) and modified using Claude Opus to work with Zabbix 7.0 LTS. Created because of my private needs. Use at your own risk.

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

<img src="https://assets.zabbix.com/img/logo/zabbix_logo_500x131.png" width="500px">

# Index

## Data Sources

* [zabbix_host](#datazabbix_host)
* [zabbix_hostgroup](#datazabbix_hostgroup)
* [zabbix_templategroup](#datazabbix_templategroup)
* [zabbix_template](#datazabbix_template)
* [zabbix_application](#datazabbix_application)
* [zabbix_proxy](#datazabbix_proxy)
* [zabbix_server](#datazabbix_server)
* [zabbix_usergroup](#datazabbix_usergroup)
* [zabbix_role](#datazabbix_role)
* [zabbix_valuemap](#datazabbix_valuemap)
* [zabbix_token](#datazabbix_token)

## Resources

* [zabbix_host](#zabbix_host)
* [zabbix_hostgroup](#zabbix_hostgroup)
* [zabbix_templategroup](#zabbix_templategroup)
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
* [zabbix_template_link](#zabbix_template_link)
* [zabbix_token](#zabbix_token)
* [zabbix_user](#zabbix_user)
* [zabbix_usergroup](#zabbix_usergroup)
* [zabbix_role](#zabbix_role)
* [zabbix_mediatype](#zabbix_mediatype)
* [zabbix_script](#zabbix_script)
* [zabbix_action](#zabbix_action)
* [zabbix_proxy](#zabbix_proxy)
* [zabbix_proxygroup](#zabbix_proxygroup)
* [zabbix_maintenance](#zabbix_maintenance)
* [zabbix_valuemap](#zabbix_valuemap)
* [zabbix_global_macro](#zabbix_global_macro)
* [zabbix_regexp](#zabbix_regexp)
* [zabbix_service](#zabbix_service)
* [zabbix_sla](#zabbix_sla)
* [zabbix_httptest](#zabbix_httptest)

# Installing and using the provider

## Locally, with no registry

Build and install it where Terraform and OpenTofu look by default:

```shell
make install            # installs local/zabbix 1.0.0 for both tools
```

Then reference it by that local address:

```hcl
terraform {
  required_providers {
    zabbix = {
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}
```

`tofu init` picks it up with no CLI configuration. Override `VERSION`,
`NAMESPACE` or `OS_ARCH` if you need a different address or platform:

```shell
make install VERSION=0.18.0 NAMESPACE=mycorp OS_ARCH=darwin_arm64
```

## For a team, with a filesystem mirror

Put the release artifacts on any shared path or object store and point the CLI
at it. This works with artifacts produced by any CI, on any git host.

```hcl
# ~/.tofurc, or a file referenced by TF_CLI_CONFIG_FILE
provider_installation {
  filesystem_mirror {
    path    = "/srv/tofu-providers"
    include = ["mycorp/*"]
  }
  direct {
    exclude = ["mycorp/*"]
  }
}
```

`tofu providers mirror /srv/tofu-providers` populates such a directory from an
existing configuration, including checksums.

## For an organisation, with a private registry

Both tools speak a documented provider registry protocol, so any implementation
of it works and none of them require GitHub. Self-hosted options include
[Terralist](https://github.com/terralist/terralist), and several CI platforms
ship a registry of their own.

## Publishing to the public registries

Both public registries read releases from GitHub:

* The **Terraform Registry** requires a public GitHub repository named
  `terraform-provider-{name}`, semver tags, and GPG-signed checksums. You sign in
  to claim the namespace with a GitHub account.
* The **OpenTofu Registry** is largely an index over GitHub releases; providers
  are submitted through an issue form on `opentofu/registry`.

So a public listing needs GitHub, but *using* the provider does not. If you
develop on another forge and still want a public listing, push-mirror the
repository and its tags to GitHub and let the release workflow there build the
artifacts. `.github/workflows/release.yml` does exactly that on tag push;
`.forgejo/workflows/` contains the equivalent CI and release pipelines for a
Forgejo instance.

# Supported Zabbix versions

| Zabbix version | Support | Notes |
|----------------|---------|-------|
| 4.0            | legacy  | untested |
| 5.0 – 6.4      | full    | untested in this fork |
| 7.0 LTS        | full    | acceptance tested against 7.0.26 |

Note on template groups: Zabbix 6.2 moved templates from host groups into
separate template groups. On Zabbix >= 6.2, `zabbix_template.groups` expects
template group IDs — use [zabbix_templategroup](#zabbix_templategroup).

# Requirements

- Access to Zabbix API over http or https

# Using the provider

## Step 1 — install

```bash
# clone the repo, then:
make install
```

Installs the provider into `~/.terraform.d/plugins` for both Terraform and
OpenTofu, with no CLI configuration required.

## Step 2 — configure

```hcl
terraform {
  required_providers {
    zabbix = {
      source  = "local/zabbix"
      version = "1.0.0"
    }
  }
}
```

Set credentials with environment variables — the simplest approach for both
development and CI:

```bash
# username + password
export ZABBIX_URL=http://zabbix.example.com/api_jsonrpc.php
export ZABBIX_USER=Admin
export ZABBIX_PASS=zabbix

# or API token (required when MFA is enabled)
export ZABBIX_URL=http://zabbix.example.com/api_jsonrpc.php
export ZABBIX_API_TOKEN=679e2404...
```

## Step 3 — use

Full working examples are in the `examples/` directory:

| File | What it shows |
|------|---------------|
| `examples/quickstart/main.tf` | Host, template, items, trigger, LLD, action, maintenance, users, token — all common resources |
| `examples/e2e/main.tf` | The same set used for automated E2E testing |
| `examples/e2e-extended/main.tf` | Extended E2E: access management, alerting, proxies, services, SLA, web monitoring |
| `examples/provider/provider.tf` | Provider block with both auth methods |

### Quickstart

```bash
git clone https://github.com/YOUR_FORK/terraform-provider-zabbix
cd terraform-provider-zabbix
make install

cd examples/quickstart
export ZABBIX_URL=http://your-zabbix/api_jsonrpc.php
export ZABBIX_USER=Admin ZABBIX_PASS=zabbix
tofu init
tofu plan
tofu apply
```

The quickstart creates a host group, template group, template with items and a
trigger, an LLD rule with item prototype, a monitored host, a webhook media
type, an action, a maintenance window, a user and an API token.


# Status

This provider is actively maintained and supports Zabbix 5.0 through 7.0 LTS.

# Testing

Acceptance tests require a running Zabbix instance. Use the included `Makefile` targets:

```bash
make test70   # run acceptance tests against Zabbix 7.0 (Docker)
make test60   # run acceptance tests against Zabbix 6.0 (Docker)
```

# Usage

All resources support terraform resource importing using zabbix ID numbers

# Templates to Terraform

The script `utils/template2terraform` provides the capabilities to convert (some of) a Zabbix XML template into Terraform HCL.

## Provider

Instantiate an instance of the provider. Authenticate either with an API token
or with a username and password.

### API token (required for MFA protected accounts)

Zabbix cannot complete an MFA challenge through `user.login`, so an account with
MFA enabled **must** use a token. The token is sent in the HTTP `Authorization`
header, which is also the only transport Zabbix 7.2 and later accept.

```hcl
provider "zabbix" {
  url       = "http://example.com/api_jsonrpc.php"
  api_token = var.zabbix_api_token
}
```

The token's role must have API access enabled. Managing global objects such as
global macros and regular expressions additionally requires a super admin role.

### Username and password

```hcl
provider "zabbix" {
  url      = "http://example.com/api_jsonrpc.php"
  username = "<api_user>"
  password = "<api_password>"
}
```

### All arguments

| Argument       | Environment                          | Description |
|----------------|--------------------------------------|-------------|
| `url`          | `ZABBIX_URL`, `ZABBIX_SERVER_URL`    | API endpoint, required |
| `api_token`    | `ZABBIX_API_TOKEN`, `ZABBIX_TOKEN`   | API token; takes precedence over username/password |
| `username`     | `ZABBIX_USER`, `ZABBIX_USERNAME`     | Required unless `api_token` is set |
| `password`     | `ZABBIX_PASS`, `ZABBIX_PASSWORD`     | Required unless `api_token` is set |
| `tls_insecure` | —                                    | Disable TLS verification, `false` by default |
| `serialize`    | —                                    | Serialize API calls, `false` by default. Enable if you hit race conditions |

Tokens can also be created by the provider itself with
[zabbix_token](#zabbix_token). Note that a configuration which manages the very
token it authenticates with cannot destroy itself, since the credential is
removed while still in use.

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

### data.zabbix_templategroup
[index](#index)

Looks up a template group by name. Template groups are a separate entity from
host groups on Zabbix >= 6.2; on older versions this falls back to the host
group API.

```hcl
data "zabbix_templategroup" "example" {
  name = "Templates/My Application"
}
```

#### Argument Reference

* name - (Required) Name of the template group

#### Attributes Reference

* name - Name of the template group

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

### zabbix_templategroup
[index](#index)

Zabbix 6.2 split template groups out of host groups into their own entity with
its own ID space. Templates must therefore be placed in a **template group**,
not a host group.

On Zabbix versions older than 6.2 this resource transparently falls back to the
host group API, so the same configuration works across versions.

```hcl
resource "zabbix_templategroup" "example" {
  name = "Templates/My Application"
}

resource "zabbix_template" "example" {
  host   = "my-application"
  groups = [zabbix_templategroup.example.id]
}
```

#### Argument Reference

* name - (Required) Name of the template group

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
* groups - (Required) List of group IDs the template belongs to. On Zabbix >= 6.2 these must be **template group** IDs (see [zabbix_templategroup](#zabbix_templategroup)); on older versions, host group IDs
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
## data.zabbix_server

Returns information about the connected Zabbix server.

```hcl
data "zabbix_server" "current" {}

output "zabbix_version" {
  value = data.zabbix_server.current.version
}
```

#### Attributes Reference

* version - Zabbix server version string (e.g. `"7.0.1"`)

## zabbix_host (tags)

The `zabbix_host` resource supports tags for Zabbix 6.0+:

```hcl
resource "zabbix_host" "web" {
  host   = "web01.example.com"
  groups = [zabbix_hostgroup.linux.id]
  interface {
    type = "agent"
    ip   = "10.0.0.1"
    port = 10050
    main = true
  }
  tag {
    key   = "env"
    value = "prod"
  }
  tag {
    key   = "team"
    value = "platform"
  }
}
```

## zabbix_template_link

Manages the contents (items, triggers, LLD rules) of a Zabbix template. This is a virtual
resource — it does not create a new object in Zabbix. It allows Terraform to declaratively
track which items and triggers belong to a template so they can be cleaned up on update.

On `terraform destroy`, the items/triggers referenced in the state are not deleted — the
template deletion itself cascades through Zabbix.

```hcl
resource "zabbix_template_link" "mytemplate" {
  template_id = zabbix_template.mytemplate.id

  item {
    item_id = zabbix_item_agent.cpu.id
  }
  item {
    item_id = zabbix_item_agent.memory.id
  }

  trigger {
    trigger_id = zabbix_trigger.cpu_high.id
  }
}
```

#### Argument Reference

* template_id - (Required, ForceNew) Template ID
* item - (Optional) Set of item IDs belonging to this template
    * item_id - (Required) Item ID
* trigger - (Optional) Set of trigger IDs belonging to this template
    * trigger_id - (Required) Trigger ID
* lld_rule - (Optional) Set of LLD rule IDs belonging to this template
    * lld_rule_id - (Required) LLD rule ID

### data.zabbix_token
[index](#index)

Looks up an existing API token by name. Exposes metadata only: the secret is
never returned by the Zabbix API and this data source deliberately has no
`token` attribute, so a secret cannot leak into state through it.

```hcl
data "zabbix_token" "ci" {
  name   = "terraform-automation"
  userid = data.zabbix_user.ci.id
}
```

#### Argument Reference

* name - (Required) Token name
* userid - (Optional) Owner of the token. Token names are only unique per user, so set this when the same name may exist for several users; the lookup errors if it matches more than one token

#### Attributes Reference

* description, enabled, expires_at, lastaccess, created_at

## zabbix_token
[index](#index)

Manages a Zabbix API token. The secret is only returned by Zabbix once, at
generation time, so it is captured on create and exported as the sensitive
`token` attribute.

```hcl
resource "zabbix_token" "automation" {
  name        = "terraform-automation"
  description = "used by CI"
  userid      = zabbix_user.automation.id
  enabled     = true
}
```

* name - (Required) Token name. Names are unique per user, not globally
* userid - (Optional, ForceNew) User the token authenticates as, defaults to the calling user
* description - (Optional) Token description
* enabled - (Optional) Whether the token may authenticate, `true` by default
* expires_at - (Optional) Unix timestamp at which the token expires, `0` for never
* token - (Computed, Sensitive) The generated secret
* lastaccess / created_at - (Computed) Unix timestamps

### Import is refused, by design

`terraform import` is rejected for this resource. An imported token would enter
state with an empty `token`, and because that attribute is computed Terraform
would report no drift — leaving a resource that looks fully managed but whose
secret is unavailable.

To adopt an existing token, pick the option that matches your intent:

* **You only need to reference it** (its id, owner or expiry): use
  [data.zabbix_token](#datazabbix_token).
* **You need a usable secret**: create a new `zabbix_token`. Regenerating the
  secret of an existing token invalidates the previous one, which would break
  whatever is already using it.

Note also that a configuration authenticating with a token it manages itself
cannot destroy that token: the credential would be removed while still in use.
Use separate credentials for such a destroy.

## zabbix_user
[index](#index)

```hcl
resource "zabbix_user" "operator" {
  username = "operator"
  name     = "Jane"
  surname  = "Doe"
  passwd   = var.operator_password
  roleid   = data.zabbix_role.admin.id
  usrgrps  = [zabbix_usergroup.ops.id]
  lang     = "en_US"
  timezone = "UTC"
}
```

A user requires at least one user group and a role. `passwd` is sensitive and is
never read back from the API.

## zabbix_usergroup
[index](#index)

```hcl
resource "zabbix_usergroup" "ops" {
  name         = "Operations"
  users_status = true

  hostgroup_rights {
    id         = zabbix_hostgroup.servers.id
    permission = "read-write"
  }
}
```

* name - (Required) Group name
* users_status - (Optional) Whether the group is enabled, `true` by default
* gui_access - (Optional) One of `default`, `internal`, `ldap`, `disabled`
* debug_mode - (Optional) Enable debug mode
* hostgroup_rights / templategroup_rights - (Optional) Permission blocks with `id` and `permission` (`read-write`, `read-only`, `deny`)

## zabbix_role
[index](#index)

```hcl
resource "zabbix_role" "automation" {
  name       = "automation"
  type       = "super_admin"
  api_access = true
}
```

* name - (Required) Role name
* type - (Required) One of `user`, `admin`, `super_admin`
* api_access - (Optional) Whether the API may be used at all. A token whose role denies API access cannot authenticate
* api_mode / api_methods - (Optional) Restrict the callable API methods
* ui / actions - (Optional) Per element blocks with `name` and `status`
* ui_default_access / actions_default_access - (Optional) Default for elements not listed

## zabbix_mediatype
[index](#index)

```hcl
resource "zabbix_mediatype" "slack" {
  name    = "Slack"
  type    = "webhook"
  script  = file("slack.js")
  timeout = "10s"

  parameters {
    name  = "url"
    value = var.slack_webhook_url
  }
}
```

`type` is one of `email`, `sms`, `script`, `webhook`; the applicable arguments
depend on it. Secrets such as `passwd` are sensitive.

## zabbix_script
[index](#index)

```hcl
resource "zabbix_script" "restart" {
  name         = "Restart service"
  type         = "script"
  scope        = "manual_host_action"
  command      = "systemctl restart myservice"
  execute_on   = "agent"
  confirmation = "Restart the service?"
  host_access  = "write"
}
```

`scope` is one of `action_operation`, `manual_host_action`,
`manual_event_action` and is immutable.

## zabbix_action
[index](#index)

```hcl
resource "zabbix_action" "notify" {
  name        = "Notify on problem"
  eventsource = "trigger"
  esc_period  = "1h"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "0"
      operator      = "0"
      value         = zabbix_hostgroup.servers.id
    }
  }

  operations {
    operationtype = "0"
    opmessage {
      default_msg = "1"
    }
    opmessage_grp {
      usrgrpid = zabbix_usergroup.ops.id
    }
  }
}
```

* eventsource - (Required, ForceNew) One of `trigger`, `discovery`, `autoregistration`, `internal`, `service`
* esc_period - (Optional) Escalation step duration. Only valid for `trigger` and `service`
* operations / recovery_operations / update_operations - Operation blocks. Escalation fields (`esc_period`, `esc_step_from`, `esc_step_to`, `evaltype`) only apply to `operations` on `trigger` actions
* filter - Condition blocks, `formulaid` only applies when `evaltype` is a custom expression

Operation types are given as the numeric Zabbix identifiers.

## zabbix_proxy
[index](#index)

```hcl
resource "zabbix_proxy" "edge" {
  name           = "edge-1"
  operating_mode = "active"
}
```

Zabbix 7.0 reshaped the proxy object: `name` replaced `host` and
`operating_mode` replaced `status`. `tls_psk` and `tls_psk_identity` are
sensitive and are not returned by the API.

## zabbix_proxygroup
[index](#index)

New in Zabbix 7.0. Proxies referencing a group must also set `local_address`.

```hcl
resource "zabbix_proxygroup" "edge" {
  name           = "edge"
  failover_delay = "1m"
  min_online     = "1"
}
```

## zabbix_maintenance
[index](#index)

```hcl
resource "zabbix_maintenance" "patching" {
  name             = "Weekly patching"
  maintenance_type = "with_data"
  active_since     = "1800000000"
  active_till      = "1830000000"
  groups           = [zabbix_hostgroup.servers.id]

  timeperiod {
    type       = "weekly"
    every      = "1"
    dayofweek  = "64"
    start_time = "7200"
    period     = "10800"
  }
}
```

* maintenance_type - (Optional) `with_data` or `no_data`. Tag filters only apply to `with_data`
* groups / hosts - At least one is required
* timeperiod - (Required) `type` is one of `one_time`, `daily`, `weekly`, `monthly`. Each type uses a different subset of the remaining fields and Zabbix rejects the ones that do not apply

## zabbix_valuemap
[index](#index)

Value maps belong to a host or template in Zabbix 6.0 and later.

```hcl
resource "zabbix_valuemap" "service_state" {
  hostid = zabbix_template.app.id
  name   = "Service state"

  mappings {
    type     = "equal"
    value    = "0"
    newvalue = "down"
  }

  mappings {
    type     = "default"
    newvalue = "unknown"
  }
}
```

## zabbix_global_macro
[index](#index)

Global macros use the `usermacro.*global` API methods, distinct from the host
level macros configured inline on `zabbix_host`.

```hcl
resource "zabbix_global_macro" "environment" {
  macro = "{$ENVIRONMENT}"
  value = "production"
}
```

Secret macros (`type = "secret"`) are never returned by the API, so the
configured value is retained rather than being read back.

## zabbix_regexp
[index](#index)

```hcl
resource "zabbix_regexp" "log_errors" {
  name        = "Log errors"
  test_string = "ERROR: disk failure"

  expressions {
    expression      = "^ERROR:"
    expression_type = "char_included"
    case_sensitive  = true
  }
}
```

## zabbix_service
[index](#index)

```hcl
resource "zabbix_service" "web" {
  name      = "Web"
  algorithm = "most_critical_child"
  sortorder = "0"

  problem_tags {
    tag      = "service"
    operator = "0"
    value    = "web"
  }
}
```

* algorithm - (Required) One of `set_ok`, `most_critical_all`, `most_critical_child`
* propagation_rule - (Optional) One of `as_is`, `increase_by`, `decrease_by`, `ignore`, `fixed`. Anything other than `as_is` requires `propagation_value`
* problem_tags - Required for a leaf service to ever enter a problem state

## zabbix_sla
[index](#index)

```hcl
resource "zabbix_sla" "web" {
  name           = "Web availability"
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
}
```

* period - (Required) One of `daily`, `weekly`, `monthly`, `quarterly`, `annually`
* status - (Optional) `enabled` or `disabled`
* service_tags - (Required) At least one entry

## zabbix_httptest
[index](#index)

```hcl
resource "zabbix_httptest" "homepage" {
  name   = "Homepage"
  hostid = zabbix_template.app.id
  delay  = "1m"

  steps {
    name         = "fetch"
    url          = "https://example.com/"
    no           = 1
    status_codes = "200"
    required     = "Example"
  }
}
```

* steps - (Required) At least one, each with a `url` and a `no` giving the order
* authentication - (Optional) One of `none`, `basic`, `ntlm`, `kerberos`, `digest`
* status - (Optional) `enabled` or `disabled`
* verify_peer / verify_host - (Optional) Booleans
* http_password - Sensitive
