# Overview

A [Terraform](terraform.io) provider for [Zabbix](https://www.zabbix.com)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

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
### zabbix_hostgroup
### zabbix_template

## Resources

### zabbix_host
### zabbix_hostgroup
### zabbix_template
### zabbix_trigger
### zabbix_item_snmp
### zabbix_item_simple
### zabbix_item_http
### zabbix_item_trapper

