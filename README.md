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
### zabbix_template

## Resources

### zabbix_host
### zabbix_hostgroup
### zabbix_template
### zabbix_trigger
### zabbix_item_agent
### zabbix_item_snmp
### zabbix_item_simple
### zabbix_item_http
### zabbix_item_trapper

