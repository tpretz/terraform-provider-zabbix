# Upgrading from v0.17.0 to v2.0.0

`v2.0.0` is the first release of this provider in a long time, and it is
deliberately a breaking one. There is no `v1.x`: the version was skipped so the
major number matches the branch the work was done on.

This guide is the whole upgrade, in the order it is least painful to do it. Read
[Before you start](#before-you-start) and
[1. Zabbix 6.0 is now the minimum](#1-zabbix-60-is-now-the-minimum) first — if
your server is older than 6.0, nothing else in here will help you and the
upgrade stops there.

Nothing in this guide needs your Zabbix objects to be recreated. Every change is
either an edit to your `.tf` files or a `terraform state` operation; monitoring
keeps running throughout.

**Summary of what breaks**

| # | Change | Effort |
|---|---|---|
| 1 | Minimum Zabbix version is **6.0** | none, if your server is 6.0+ |
| 2 | `zabbix_application` and `data.zabbix_application` removed | edit + `terraform state rm` |
| 3 | `applications` removed from every item | delete the attribute |
| 4 | `zabbix_item_aggregate` / `zabbix_proto_item_aggregate` removed | rewrite as calculated items + `terraform state rm` |
| 5 | `zabbix_template.groups` takes **template** group ids on 6.2+ | edit + `terraform import` |
| 6 | `graph.item`, `host.interface`, LLD `condition` are **sets, not lists** | fix every `[0]` index |
| 7 | Legacy SNMP item attributes removed | delete the attributes |
| 8 | `preprocessor.type` takes a name, not a number | none now, edit at leisure |
| 9 | `serialize` now defaults to `true` | none — a behaviour change, not a config one |
| 10 | `inventory` under `inventory_mode = "automatic"` no longer clears fields you stop naming | none, unless you were relying on it |

Numbers 9 and 10 need no action at all: applies get safer, and marginally slower,
on their own, and number 10 stops the provider deleting inventory data Zabbix
collected for you. Numbers 3 and 7 are the cheapest: the attributes are simply deleted from your
config and the provider quietly discards them from state. Number 8 is cheaper
still — nothing to do at upgrade time. Number 6 is the one most likely to break
a config that otherwise looks fine.

---


## Before you start

Pin the new version explicitly rather than letting a constraint float into it:

```hcl
terraform {
  required_providers {
    zabbix = {
      source  = "tpretz/zabbix"
      version = "~> 2.0"
    }
  }
}
```

Take a copy of your state before the first `terraform plan` under v2:

```bash
terraform state pull > terraform.tfstate.v0.17.0.backup
```

State written by v2 cannot be read by v0.17.0 again — the schema versions have
moved on. The backup is your only way back.

Then work through the sections below **before** running `terraform apply`. A
`terraform plan` is safe at any point and is the fastest way to find what you
have missed; several of these changes surface as ordinary configuration errors.

---

## 1. Zabbix 6.0 is now the minimum

Support for Zabbix 4.0, 5.0 and 5.4 has been **deleted**, not merely left
untested. The provider is tested against 6.0 LTS, 7.0 LTS and 7.4, with 8.0
watched as an early warning.

If you are on an older server, upgrade Zabbix first. Everything below assumes
6.0 or later.

The flip side: v0.17.0 could not talk to a Zabbix **7.2 or later** server at all
— it sent the auth token as a JSON-RPC `auth` property, which 7.2 removed and
now rejects outright, failing every single call. If you are on 7.2+, v2.0.0 is
not optional.

---

## 2. `zabbix_application` is gone — use item tags

Zabbix removed applications in **5.4** and replaced them with item tags. The
resource and data source have been removed with them.

**Before**

```hcl
resource "zabbix_application" "cpu" {
  name   = "CPU"
  hostid = zabbix_host.web.id
}

data "zabbix_application" "memory" {
  name   = "Memory"
  hostid = zabbix_host.web.id
}
```

**After**

There is no replacement resource. An application was a grouping of items, and
that grouping is now expressed on the items themselves:

```hcl
resource "zabbix_item_agent" "cpu_load" {
  hostid    = zabbix_host.web.id
  key       = "system.cpu.load[all,avg1]"
  name      = "CPU load (1m avg)"
  valuetype = "float"

  tag {
    key   = "component"
    value = "cpu"
  }
}
```

`component` is Zabbix's own convention for this and is what the bundled
templates use, but the key is free-form — any `key`/`value` pair works.

### Removing them from state

There is no upgrade path for a resource that no longer exists. Terraform will
refuse to do anything at all while `zabbix_application` instances are in state,
with an error like:

```
Error: Invalid resource type
  provider registry.terraform.io/tpretz/zabbix does not support resource type "zabbix_application"
```

Delete the `resource` and `data` blocks from your configuration, then forget the
instances:

```bash
terraform state rm zabbix_application.cpu
```

`terraform state rm` only forgets the object; it does **not** delete anything in
Zabbix. On a 6.0+ server the application rows are long gone anyway.

---

## 3. `applications` is gone from every item

Every `zabbix_item_*` and `zabbix_proto_item_*` resource had an `applications`
attribute. It is removed.

**Before**

```hcl
resource "zabbix_item_agent" "cpu_load" {
  hostid       = zabbix_host.web.id
  key          = "system.cpu.load[all,avg1]"
  name         = "CPU load (1m avg)"
  valuetype    = "float"
  applications = [zabbix_application.cpu.id]
}
```

**After**

```hcl
resource "zabbix_item_agent" "cpu_load" {
  hostid    = zabbix_host.web.id
  key       = "system.cpu.load[all,avg1]"
  name      = "CPU load (1m avg)"
  valuetype = "float"

  tag {
    key   = "component"
    value = "cpu"
  }
}
```

Leaving it in the configuration is a hard error:

```
Error: Unsupported argument
  An argument named "applications" is not expected here.
```

**You do not need to do anything to your state.** Terraform hands the provider
the old state and the provider discards attributes its schema no longer
declares, silently and on the first plan. There is nothing to import, no
`terraform state rm`, and no `-refresh-only` dance. Delete the attribute from
the configuration and carry on.

---

## 4. Aggregate items are gone — use calculated items

Zabbix removed the aggregate item **type** in 5.4. `zabbix_item_aggregate` and
`zabbix_proto_item_aggregate` are removed with it. The functionality did not go
away: it moved into calculated items, as *foreach* functions over an item
filter.

**Before**

```hcl
resource "zabbix_item_aggregate" "cluster_load" {
  hostid    = zabbix_host.cluster.id
  key       = "grpavg[\"web servers\",\"system.cpu.load[all,avg1]\",last]"
  name      = "Cluster CPU load"
  valuetype = "float"
  delay     = "1m"
}
```

**After**

```hcl
resource "zabbix_item_calculated" "cluster_load" {
  hostid    = zabbix_host.cluster.id
  key       = "cluster.cpu.load"
  name      = "Cluster CPU load"
  valuetype = "float"
  delay     = "1m"

  formula = "avg(last_foreach(/*/system.cpu.load[all,avg1]?[group=\"web servers\"]))"
}
```

Note that the **key changes**. Under the aggregate type the key *was* the
calculation; a calculated item has a key you choose plus a separate `formula`.
Pick a key that reads well — it is what triggers and graphs will reference.

The translation is mechanical:

| Aggregate key | Calculated formula |
|---|---|
| `grpavg["G","key",last]` | `avg(last_foreach(/*/key?[group="G"]))` |
| `grpsum["G","key",last]` | `sum(last_foreach(/*/key?[group="G"]))` |
| `grpmin["G","key",last]` | `min(last_foreach(/*/key?[group="G"]))` |
| `grpmax["G","key",last]` | `max(last_foreach(/*/key?[group="G"]))` |
| `grpavg["G","key",avg,5m]` | `avg(avg_foreach(/*/key?[group="G"],5m))` |
| `grpavg["G","key",count]` | `avg(count_foreach(/*/key?[group="G"]))` |

Multiple groups become an `or`: `?[group="A" or group="B"]`.

### Removing them from state

As with `zabbix_application`, there is no upgrade path — the resource type does
not exist. Because the replacement is a *different resource type* with a
*different key*, you cannot import the old item into the new resource either;
the old aggregate item cannot be created on a 6.0+ server in the first place.

```bash
terraform state rm zabbix_item_aggregate.cluster_load
```

Then `terraform apply` to create the calculated item. If the old aggregate item
still physically exists on the server (it can, if the row survived a Zabbix
upgrade), delete it from the Zabbix frontend — Terraform has forgotten it.

---

## 5. `zabbix_template.groups` now means *template* groups

Zabbix **6.2** split template groups out of host groups. They are now two
separate object types with two separate id spaces, and `template.create` /
`template.update` reject a host group id.

The provider follows the server: on 6.2 and later `zabbix_template.groups` holds
**template** group ids; on 6.0 and 6.1 it still holds host group ids, and
`zabbix_templategroup` errors if you try to use it.

**Before**

```hcl
resource "zabbix_hostgroup" "templates" {
  name = "Templates/Applications"
}

resource "zabbix_template" "base" {
  host   = "tf-linux-base"
  name   = "TF Linux Base"
  groups = [zabbix_hostgroup.templates.id]
}
```

**After** (Zabbix 6.2+)

```hcl
resource "zabbix_templategroup" "templates" {
  name = "Templates/Applications"
}

resource "zabbix_template" "base" {
  host   = "tf-linux-base"
  name   = "TF Linux Base"
  groups = [zabbix_templategroup.templates.id]
}
```

If the group already exists on the server — which it usually does, since Zabbix
created it during its own 6.2 upgrade — use the data source instead of declaring
a resource you do not own:

```hcl
data "zabbix_templategroup" "templates" {
  name = "Templates/Applications"
}

resource "zabbix_template" "base" {
  host   = "tf-linux-base"
  name   = "TF Linux Base"
  groups = [data.zabbix_templategroup.templates.id]
}
```

### Why the provider will not fix the ids for you

`zabbix_template` has a state upgrader, and it **verifies rather than rewrites**.
Ids that already resolve via `templategroup.get` pass through untouched;
anything else stops the upgrade with an error naming the offending id.

That is deliberate, and it is not laziness. Zabbix's own 6.2 database upgrade
did one of two different things to each host group:

- a group containing **only templates** was converted into a template group
  **in place, keeping its id** — so your state is already correct;
- a **mixed** group containing hosts *and* templates was left as a host group
  and a **brand new template group with a fresh id** was created alongside it.

Nothing in the state file distinguishes those two cases, and an operator may
since have renamed, merged or deleted either side. Any mechanical translation
would be a guess, and silently pointing a template at the wrong group is a much
worse outcome than refusing to proceed. So the provider refuses, and tells you
which id it could not account for:

```
Error: zabbix_template "tf-linux-base": `groups` holds ids that are not template
groups on this server: 16 (host group "Templates/Applications").
```

The host group name in that message is the clue you need: the template group
Zabbix created will almost always have the same name.

### Fixing state

If the ids are already correct (the converted-in-place case) the upgrade passes
and there is nothing to do. Otherwise, for each affected template:

1. Change the configuration to reference a `zabbix_templategroup` resource or
   data source, as above.
2. Forget the template and re-import it, so its state is rebuilt from the
   server:

   ```bash
   terraform state rm zabbix_template.base
   terraform import zabbix_template.base 10500
   ```

   The id is the Zabbix `templateid`, which you can read off the template's URL
   in the frontend, or out of your state backup.

3. `terraform plan` and confirm it is empty.

Re-importing is safer than hand-editing the id in the state file, because the
import reads every attribute back from the server rather than trusting the rest
of the stale record.

`terraform state rm` does not delete the template or anything linked to it.

### Still on 6.0 or 6.1?

Keep using `zabbix_hostgroup` for template groups. `zabbix_templategroup` will
error, and the state upgrader detects the server version and passes your groups
through unchanged. Revisit this section when you upgrade to 6.2+.

---

## 6. Sets, not lists — and sets cannot be indexed

**This is the change most likely to break a configuration that otherwise looks
fine.** Three collections that were `TypeList` are now `TypeSet`:

| Resource | Block |
|---|---|
| `zabbix_graph`, `zabbix_proto_graph` | `item` |
| `zabbix_host` | `interface` |
| every `zabbix_lld_*` | `condition` (the LLD filter) |

The reason is that Zabbix does not return any of them in a stable order, and the
order is not even stable *across versions*: 8.0 reorders graph items, and 7.2
returns LLD filter conditions sorted by formula id where 6.0 returned them in
submission order. Modelled as a list, that produced permanent spurious diffs —
Terraform saw a reordering as a change and planned an update on every run. As a
set, ordering is not part of the value and the diffs go away.

Writing the blocks does not change at all:

```hcl
resource "zabbix_host" "web" {
  host   = "web01.example.com"
  groups = [zabbix_hostgroup.linux.id]

  interface {
    type = "agent"
    ip   = "10.0.0.11"
  }

  interface {
    type = "snmp"
    ip   = "10.0.0.11"
  }
}
```

**Reading them does.** A set has no indices, so any expression that used one is
now an error:

```
Error: Invalid index
  This value does not have any indices.
```

### Fixing references

**One element — use `one()`**

```hcl
# Before
interfaceid = zabbix_host.web.interface[0].id

# After
interfaceid = one(zabbix_host.web.interface).id
```

`one()` returns the single element of a collection, or `null` if it is empty. It
*errors* if there is more than one, which is the behaviour you want: it turns a
wrong assumption into a loud failure rather than a silently arbitrary pick.

**More than one element — select the one you mean**

```hcl
# Before
interfaceid = zabbix_host.web.interface[1].id   # the SNMP one, by position

# After
interfaceid = one([
  for i in zabbix_host.web.interface : i if i.type == "snmp"
]).id
```

Or, if you reference it more than once:

```hcl
locals {
  web_ifaces = { for i in zabbix_host.web.interface : i.type => i }
}

resource "zabbix_item_snmp" "if_in" {
  hostid      = zabbix_host.web.id
  interfaceid = local.web_ifaces["snmp"].id
  # ...
}
```

**Do not** reach for `tolist(zabbix_host.web.interface)[0]`. It compiles and it
will usually appear to work, but a set's iteration order is by internal hash,
not by the order you wrote the blocks — so which interface you get is arbitrary
and can change when an unrelated attribute changes. If your old `[0]` meant "the
agent interface", say that.

The same three fixes apply to `zabbix_graph.x.item[0]` and to
`zabbix_lld_agent.x.condition[0]`.

### State

Nothing to do. All three resources declare a state upgrader that reads your
prior state in its old list shape and hands it to the new set schema, on both
the modern JSON state format and the flatmap format written by Terraform 0.11.
It is applied automatically on the first plan.

### While you are here: LLD filter formula ids

With `evaltype = "custom"` you now set `id` on each `condition` block yourself
and reference those ids from `formula`. Under every other evaltype Zabbix
assigns the ids, 7.2+ rejects a caller-supplied value, and the attribute stays
empty — so if you previously read `condition[*].id` back out of state for
anything, it will now be empty unless the evaltype is `custom`.

```hcl
resource "zabbix_lld_agent" "mounts" {
  hostid   = zabbix_host.web.id
  key      = "vfs.fs.discovery"
  name     = "Mounted filesystem discovery"
  evaltype = "custom"
  formula  = "A and B"

  condition {
    id    = "A"
    macro = "{#FSTYPE}"
    value = "ext4|xfs"
  }

  condition {
    id       = "B"
    macro    = "{#FSNAME}"
    value    = "^/run"
    operator = "notmatch"
  }
}
```

---

## 7. Legacy SNMP item attributes are gone

Zabbix 5.0 collapsed the three SNMP item types into one and moved the SNMP
credentials from the item onto the **host interface**, where they had always
belonged. With the 6.0 floor, the provider has dropped the item-level copies.

Removed from `zabbix_item_snmp`, `zabbix_proto_item_snmp` and `zabbix_lld_snmp`:
`snmp_version`, `snmp_community`, `snmp3_authpassphrase`, `snmp3_authprotocol`,
`snmp3_contextname`, `snmp3_privpassphrase`, `snmp3_privprotocol`,
`snmp3_securitylevel`, `snmp3_securityname`. Only `snmp_oid` remains.

**Before**

```hcl
resource "zabbix_item_snmp" "if_in" {
  hostid         = zabbix_host.web.id
  interfaceid    = zabbix_host.web.interface[1].id
  key            = "ifHCInOctets[1]"
  name           = "Interface 1: bits in"
  valuetype      = "unsigned"
  snmp_oid       = "1.3.6.1.2.1.31.1.1.1.6.1"
  snmp_version   = "2"
  snmp_community = "{$SNMP_COMMUNITY}"
}
```

**After**

```hcl
resource "zabbix_host" "web" {
  host   = "web01.example.com"
  groups = [zabbix_hostgroup.linux.id]

  interface {
    type           = "snmp"
    ip             = "10.0.0.11"
    snmp_version   = "2"
    snmp_community = "{$SNMP_COMMUNITY}"
  }
}

resource "zabbix_item_snmp" "if_in" {
  hostid = zabbix_host.web.id
  interfaceid = one([
    for i in zabbix_host.web.interface : i if i.type == "snmp"
  ]).id
  key       = "ifHCInOctets[1]"
  name      = "Interface 1: bits in"
  valuetype = "unsigned"
  snmp_oid  = "1.3.6.1.2.1.31.1.1.1.6.1"
}
```

Note that this example also needs the [set fix](#6-sets-not-lists--and-sets-cannot-be-indexed)
— an SNMP item is the most common place to find an `interface[N]` index.

As with `applications`, the leftover attributes in **state** need no action; the
provider discards them on the first plan. Only the configuration has to change.

---

## 8. `preprocessor.type` takes a name, not a number

`preprocessor.type` used to be Zabbix's internal numeric code, validated only
as "must be numeric". It is now a named enum, like every other enum in the
provider:

**Before**

```hcl
resource "zabbix_item_agent" "queue" {
  hostid    = zabbix_host.web.id
  key       = "app.queue"
  name      = "Queue depth"
  valuetype = "unsigned"

  preprocessor {
    type   = "12"
    params = ["$.depth"]
  }

  preprocessor {
    type   = "20"
    params = ["1h"]
  }
}
```

**After**

```hcl
resource "zabbix_item_agent" "queue" {
  hostid    = zabbix_host.web.id
  key       = "app.queue"
  name      = "Queue depth"
  valuetype = "unsigned"

  preprocessor {
    type   = "jsonpath"
    params = ["$.depth"]
  }

  preprocessor {
    type   = "discard_unchanged_heartbeat"
    params = ["1h"]
  }
}
```

### Nothing breaks if you do not do this

The numeric form is still accepted. It emits a deprecation warning on every
plan and **will be removed in the next major release**, but it applies, and it
converges on its own: the provider rewrites the number to the name in state on
the first apply, so the plan after that is empty and the configuration keeps
working unchanged until you get round to it.

Concretely, upgrading a state written by v0.17.0 shows one in-place update per
item that has preprocessing — `type` changing from the number to the name, with
nothing sent to Zabbix that it did not already have. Apply it and the diff is
gone for good.

### The names

The number in brackets is Zabbix's code, so an old configuration can be
translated by looking it up.

| Item | | |
|---|---|---|
| `multiplier` (1) | `rtrim` (2) | `ltrim` (3) |
| `trim` (4) | `regex` (5) | `bool_to_decimal` (6) |
| `octal_to_decimal` (7) | `hex_to_decimal` (8) | `simple_change` (9) |
| `change_per_second` (10) | `xml_xpath` (11) | `jsonpath` (12) |
| `in_range` (13) | `matches_regex` (14) | `not_matches_regex` (15) |
| `check_json_error` (16) | `check_xml_error` (17) | `check_regex_error` (18) |
| `discard_unchanged` (19) | `discard_unchanged_heartbeat` (20) | `javascript` (21) |
| `prometheus_pattern` (22) | `prometheus_to_json` (23) | `csv_to_json` (24) |
| `replace` (25) | `check_unsupported` (26) | `xml_to_json` (27) |
| `snmp_walk_value` (28) | `snmp_walk_to_json` (29) | `snmp_get_value` (30) |

A **discovery rule** takes a smaller list, and always did — Zabbix refuses the
rest on a `zabbix_lld_*` resource. The provider now refuses them too, at plan
time, instead of passing them through for the server to reject:

`regex` (5), `xml_xpath` (11), `jsonpath` (12), `matches_regex` (14),
`not_matches_regex` (15), `check_json_error` (16), `check_xml_error` (17),
`discard_unchanged_heartbeat` (20), `javascript` (21), `prometheus_to_json`
(23), `csv_to_json` (24), `replace` (25), `xml_to_json` (27),
`snmp_walk_value` (28), `snmp_walk_to_json` (29), `snmp_get_value` (30).

### Types your server may not have

Four of these arrived after 6.0, and the provider now says so at apply time
rather than letting the server answer with a number you did not write:

| Type | Needs |
|---|---|
| `snmp_walk_value` (28) | Zabbix 6.4 |
| `snmp_walk_to_json` (29) | Zabbix 6.4 |
| `snmp_get_value` (30) | Zabbix 7.0 |
| `matches_regex` (14) **on a discovery rule only** | Zabbix 7.0 |

`matches_regex` is the odd one: on an *item* it has existed since 6.0 and is
not gated. Only the discovery-rule use of it needs 7.0.

### While you are here: `params` no longer rejects an empty string

Related, and a fix rather than a break. A step's parameters are positional
slots, and an empty one is a real value — `prometheus_pattern` with output
`value` is stored by Zabbix as three parameters with the third empty, whatever
you send. The provider used to reject `""` at plan time, which made that step
impossible to write in a form that matched what the server returned:

```hcl
preprocessor {
  type   = "prometheus_pattern"
  params = ["cpu_usage_system", "value", ""]
}
```

If you have a `prometheus_pattern` step that has been showing a permanent diff,
adding the empty third parameter is the fix.

---

## 9. `serialize` now defaults to `true`

`v0.17.0` sent API requests concurrently, following Terraform's parallelism. `v2.0.0`
sends **mutating** requests one at a time by default. Reads are untouched, so `plan` and
`refresh` are exactly as fast as before; a full acceptance run costs roughly 3% more.

This is a workaround for concurrency bugs in Zabbix, not a tuning decision. Two distinct
failures have been observed against real servers:

- A host linked to a template kept the template's items and lost every one of its
  triggers. Nothing reported an error. It surfaced weeks later, as
  `Database error occurred` on an unrelated change that happened to depend on the
  missing trigger.
- A parallel `terraform destroy` failed with
  `duplicate key value violates unique constraint "ids_pkey"` on
  `(housekeeper, housekeeperid)`. Zabbix allocates internal ids with a
  `SELECT ... FOR UPDATE` followed by an `INSERT`, which two concurrent transactions
  can both reach.

Both are Zabbix-side races that the provider cannot detect and cannot repair, and the
first is silent — which is why the safe setting is the default one.

**It only protects a single `terraform apply`.** The lock lives in one provider process.
Two applies running at once, or Terraform racing a change someone makes in the Zabbix
UI, will still collide.

To restore the old behaviour:

```hcl
provider "zabbix" {
  serialize = false
}
```

Only do that if you are confident your configuration cannot race — in practice, that it
does not link several hosts to one template.

## 10. `inventory` under `inventory_mode = "automatic"` leaves unnamed fields alone

Nothing to edit, and for most configurations nothing changes. It is listed because
it changes what an apply *does not* do.

Under `inventory_mode = "automatic"` Zabbix populates inventory fields itself, from
any item carrying an `inventory_link`. `v0.17.0` through the v2 pre-release copied
every field the server returned into state and then sent `""` for anything state
held that the configuration no longer named — which is exactly the set Zabbix had
filled in. The result, verified on 8.0:

```
  ~ inventory {
      - os = "Linux 6.1 (auto-discovered)" -> null
    }
```

and the apply wiped it on the server. Zabbix repopulated it, the next plan proposed
the deletion again, and the discovered data was the casualty of a fight that never
ended.

From `v2.0.0`, under **automatic** mode the provider sends exactly the fields your
`inventory` block names and leaves every other one alone:

```hcl
resource "zabbix_host" "example" {
  # ...
  inventory_mode = "automatic"

  inventory {
    name     = "example.internal"   # sent, and managed
    location = ""                   # sent as "", so it is cleared
    # os is not named, so Zabbix owns it: not read into state, never deleted
  }
}
```

**Manual mode is unchanged.** There the configuration owns the whole inventory, and
deleting a line still clears the field on the server.

The cost, and it is the only place in this provider where deleting a line does not
revert an attribute: under automatic mode, **removing a line leaves the field as it
was**. Write it as `""` to clear it. Do that while the field is still in your block —
once you have deleted the line, the field has also left Terraform's state, so adding
`name = ""` back is not a change Terraform can see and it clears nothing. Writing any
other value works normally, and `inventory_mode = "manual"` gets you the old
behaviour for the whole block.

## Checklist

```
[ ] Zabbix server is 6.0 or later
[ ] required_providers pinned to ~> 2.0
[ ] state backed up (terraform state pull > backup)
[ ] zabbix_application resources/data sources deleted from config
[ ] terraform state rm for every zabbix_application
[ ] applications = [...] deleted from every item
[ ] zabbix_item_aggregate rewritten as zabbix_item_calculated
[ ] terraform state rm for every zabbix_item_aggregate
[ ] zabbix_template.groups points at zabbix_templategroup (6.2+)
[ ] templates with stale group ids re-imported
[ ] every interface[N] / item[N] / condition[N] index replaced
[ ] legacy snmp_* attributes moved from items to host interfaces
[ ] preprocessor { type = "<number>" } rewritten to the name (optional, but
    the numeric form goes away in the next major release)
[ ] serialize left at its new default of true, unless you know you need
    otherwise (nothing to change — noted so it is a decision, not a surprise)
[ ] inventory blocks under inventory_mode = "automatic" reviewed: a field you
    want cleared is now written as "" rather than deleted (nothing to change
    unless you were relying on deletion clearing it)
[ ] terraform plan is empty
```

A clean, empty `terraform plan` is the finish line. If it is not empty and you
cannot see why, `TF_LOG=TRACE terraform plan` shows the provider's own
`[TRACE]`/`[DEBUG]` lines, including the exact API calls being made.

Anything this guide does not cover, or covers wrongly, is a bug — please open an
issue.
