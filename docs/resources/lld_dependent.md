---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "zabbix_lld_dependent Resource - terraform-provider-zabbix"
subcategory: ""
description: |-
  
---

# zabbix_lld_dependent (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **hostid** (String) Host ID
- **key** (String) LLD KEY
- **master_itemid** (String) Master Item ID
- **name** (String) LLD Name

### Optional

- **condition** (Block List) (see [below for nested schema](#nestedblock--condition))
- **delay** (String) LLD Delay period
- **evaltype** (String) EvalType, one of: or, custom, andor, and
- **formula** (String) Formula
- **id** (String) The ID of this resource.
- **lifetime** (String) LLD Stale Item Lifetime
- **macro** (Block Set) (see [below for nested schema](#nestedblock--macro))
- **preprocessor** (Block List) (see [below for nested schema](#nestedblock--preprocessor))

<a id="nestedblock--condition"></a>
### Nested Schema for `condition`

Required:

- **macro** (String) Filter Macro
- **value** (String) Filter Value

Optional:

- **operator** (String) Operator, one of: match, notmatch

Read-Only:

- **id** (String) The ID of this resource.


<a id="nestedblock--macro"></a>
### Nested Schema for `macro`

Required:

- **macro** (String) Macro
- **path** (String) Macro Path


<a id="nestedblock--preprocessor"></a>
### Nested Schema for `preprocessor`

Required:

- **type** (String) Preprocessor type, zabbix identifier number

Optional:

- **error_handler** (String)
- **error_handler_params** (String)
- **params** (List of String) Preprocessor parameters

Read-Only:

- **id** (String) The ID of this resource.


