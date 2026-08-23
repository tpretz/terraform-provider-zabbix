# Changelog

All notable changes to this provider are documented in this file.

The format is [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is maintained by hand: add your entry to `## [Unreleased]` in the same
commit as the change. There is deliberately no changelog-fragment tool — see
[CONTRIBUTING.md § Changelog entries](./CONTRIBUTING.md#changelog-entries) for why,
and for the one condition that should change that decision.

Releases before `2.0.0` predate this file. Their history is the git log and the
[GitHub releases](https://github.com/tpretz/terraform-provider-zabbix/releases)
page. There is no `1.x`: the major version was skipped so that the branch name,
the module major and the Registry version all read the same.

## [Unreleased]

Nothing yet.

---

## [2.0.0] — UNRELEASED

The first release since `v0.17.0` (2021), and deliberately a breaking one. Every
breaking change in the v2 line is batched here.

**Read [MIGRATING.md](./MIGRATING.md) before upgrading.** It is a section-by-section
`v0.17.0 → v2.0.0` guide with before/after HCL for each of its twelve numbered
sections — eight that need an edit, four that only change what an apply does —
and it ends in a checklist. No Zabbix object has to be recreated: every
change is either an edit to your `.tf` files or a `terraform state` operation.

Headline: **the provider could not talk to a Zabbix 7.2 or newer server at all**
before this release — the auth token was sent as a JSON-RPC `auth` body property,
which 7.2 removed and now rejects, so every single API call failed. That, and the
other thirty-four defects fixed along the way, are listed under
[Fixed](#fixed) below.

Supported Zabbix versions: **6.0 LTS, 7.0 LTS and 7.4** are release-gating; 8.0 is
watched non-blocking via the `ubuntu-trunk` nightly until it reaches GA.

### Added

- `zabbix_templategroup` resource and data source. Zabbix 6.2 split template
  groups out of host groups; below 6.2 both refuse with an actionable error
  pointing at `zabbix_hostgroup`.
- `zabbix_proxy` **resource**. The provider could previously look a proxy up but
  not manage one. It exposes one set of attribute names on every version and
  translates internally across Zabbix 7.0's rewrite of the object
  (`host`→`name`, `status`→`operating_mode`, nested interface→`address`/`port`,
  `proxy_address`→`allowed_addresses`).
- `token` provider argument (`$ZABBIX_TOKEN`), as an alternative to
  `username`/`password`. With a token the provider skips the login call. From
  Zabbix 6.4 the token is sent as an `Authorization: Bearer` header; below that,
  as the JSON-RPC body property.
- `username` and `password` are now optional, with `$ZABBIX_USER`/`$ZABBIX_USERNAME`
  and `$ZABBIX_PASS`/`$ZABBIX_PASSWORD` fallbacks. One of the two credential paths
  is still required, and the provider says so rather than failing at the first call.
- `zabbix_host`: `ipmi_authtype`, `ipmi_privilege`, `ipmi_username`,
  `ipmi_password`, and `tls_connect`, `tls_accept`, `tls_issuer`, `tls_subject`,
  `tls_psk_identity`, `tls_psk`. Password and PSK attributes are marked sensitive.
- `zabbix_template`: `uuid` (computed), `vendor_name`/`vendor_version` (Zabbix 6.4+),
  `readme`/`wizard_ready` (7.4+). Below their gate the attributes are absent rather
  than erroring.
- `zabbix_trigger`: `event_name`, `opdata` and `correlation_mode`.
  `correlation_mode` is now settable in its own right; `v0.17.0` inferred it
  from a non-empty `correlation_tag` and offered no way to say "tag" without
  one, or "disabled" with one. (`manual_close`, `correlation_tag`,
  `dependencies` and `tag` already existed in `v0.17.0`.)
- `units` and `description` on every item and item prototype resource — all ten
  `zabbix_item_*` and all ten `zabbix_proto_item_*`. Without `units` every item
  rendered as a bare number in the frontend, which was the provider's most
  conspicuous omission. `uptime` and `unixtime` get Zabbix's special formatting
  and a leading `!` suppresses the automatic multiplier; both are documented on
  the attribute. Note that from Zabbix 7.0 a non-empty `units` is rejected on a
  character, log or text item — 6.0 accepts one on any value type.
- `description` on every `zabbix_lld_*` discovery rule. `units` deliberately does
  **not** appear there: `discoveryrule.get` returns the property on every version,
  but `discoveryrule.create`/`.update` reject it from 7.0 with
  `unexpected parameter "units"`.
- Generated Registry documentation. `docs/` is now produced by `terraform-plugin-docs`
  from the schema, `templates/` and `examples/` — 42 pages, each with a runnable
  example and an import script, grouped into seven Registry subcategories. Every
  one of the 976 schema attributes now carries a description, enforced by a unit
  test.
- [MIGRATING.md](./MIGRATING.md), [TESTING.md](./TESTING.md),
  [DEVELOPMENT.md](./DEVELOPMENT.md), [CONTRIBUTING.md](./CONTRIBUTING.md),
  [RELEASING.md](./RELEASING.md) and [MAINTAINING.md](./MAINTAINING.md).

### Changed

- **`serialize` now defaults to `true`**, and applies only to mutating requests.
  Reads are never serialized, so `plan` and `refresh` are unchanged and a full
  acceptance run costs about 3% more. This is a workaround for concurrency bugs in
  Zabbix rather than a tuning default: template inheritance and internal id
  allocation both race under Terraform's default parallelism of 10. Two failures
  observed against real servers — a host that kept a template's items and silently
  lost every one of its triggers, and a parallel destroy failing with
  `duplicate key value violates unique constraint "ids_pkey"` on
  `(housekeeper, housekeeperid)`. Note the lock is per provider process, so it
  protects a single `terraform apply` and nothing wider. See MIGRATING.md § 9.

- **Minimum Zabbix version is 6.0.** 4.0, 5.0 and 5.4 support was deleted, not
  merely left untested. See [MIGRATING.md §1](./MIGRATING.md#1-zabbix-60-is-now-the-minimum).
- **`graph.item`, `zabbix_host.interface`, the LLD filter `condition` block and
  `macro` are sets, not lists.** The server's return order for all four is an
  implementation detail that changed between Zabbix versions. Sets cannot be
  indexed from HCL: `zabbix_host.x.interface[0].id` no longer parses — use
  `one(...)`. This is the change most likely to break a configuration that
  otherwise looks fine. See
  [MIGRATING.md §6](./MIGRATING.md#6-sets-not-lists--and-sets-cannot-be-indexed).
  `macro` is affected on `zabbix_host` and `zabbix_template` and on both data
  sources. State upgraders are provided on all twelve affected resources; item
  and LLD `preprocessing` deliberately stay lists, because their order is
  semantic.
- **`zabbix_template.groups` means *template* groups on Zabbix 6.2+.** A state
  upgrader verifies the ids rather than rewriting them: a host group id cannot be
  mechanically turned into a template group id, so anything that is not already a
  template group fails naming the offending ids and the fix. See
  [MIGRATING.md §5](./MIGRATING.md#5-zabbix_templategroups-now-means-template-groups).
- **Defaults that are derived rather than fixed are now worked out at plan time.**
  Four attributes have a default the schema cannot state as a constant, because it
  follows another attribute: `zabbix_host.name` and `zabbix_template.name` (from
  `host`), item `trends` (from `valuetype`) and trigger `correlation_mode` (from
  `correlation_tag`). They used to be left to the apply, so the plan said
  `(known after apply)` for a value written three lines further up. Each is now
  derived in `CustomizeDiff`, and each firing condition is deliberately narrow so
  that nothing the user owns is overwritten: the two names on create and on a
  `host` rename *while the stored name is still the old `host`*, `trends` on
  create and on a `valuetype` change across the numeric boundary,
  `correlation_mode` on create only (where it keeps inferring "tag" from a
  `correlation_tag` written on its own, the shape configurations predating the
  attribute use). Deleting any of the four lines still changes
  nothing, exactly as before. The one visible behaviour change: renaming `host` on
  a resource whose `name` was derived now moves the display name along with it,
  where it used to be left pointing at the old technical name for ever. A display
  name you set yourself is never touched. See
  [MIGRATING.md §11](./MIGRATING.md#11-renaming-host-now-moves-the-display-name-with-it).
- **`zabbix_host.interface` is `Optional` rather than `Required`.** Zabbix
  accepts a host with no interfaces at all on every supported version — one
  carrying only calculated, dependent, trapper or internal items, or existing
  purely to hold templates, has nothing to attach an interface to. Nothing to
  change in an existing configuration; it only stops the provider refusing a
  host the server would have created. See
  [MIGRATING.md §12](./MIGRATING.md#12-smaller-changes-that-need-no-action).
- The Zabbix API client is no longer the `github.com/tpretz/go-zabbix-api` git
  submodule; it lives in this repository as `internal/zabbix`, with its history
  preserved. Nothing about this is visible in a configuration.
- Toolchain: Go directive `1.25.8`, `terraform-plugin-sdk/v2` v2.40.1,
  `terraform-plugin-go` v0.31.0. The acceptance suite moved to
  `terraform-plugin-testing` v1.16.0. `.tool-versions` pins the versions actually
  built and tested with (Go 1.25.12, Terraform 1.8.5).
- The acceptance harness is a root-level `docker-compose.test.yml` running plain
  `docker compose` — one isolated PostgreSQL-backed stack per Zabbix version, with
  healthchecks that wait on `apiinfo.version` rather than on the web root. The old
  devcontainer-only 4.0/5.0/5.4/6.0 MySQL harness is gone.
- Release plumbing: `.goreleaser.yml` migrated to the v2 schema, builds narrowed to
  amd64/arm64 across freebsd/windows/linux/darwin, and the archived
  `hashicorp/ghaction-import-gpg` (a Node 12 action GitHub no longer executes at
  all) replaced with `crazy-max/ghaction-import-gpg@v6`.

### Deprecated

- `data.zabbix_proxy`'s `host` argument. Use `name`, which is what the object has
  been called since Zabbix 7.0. Both are still accepted and both are reported.

### Fixed

Every defect below landed as its own reviewed change. Almost all were found by
tests written during the v2 work rather than reported, and several made a
resource completely unusable on a current Zabbix server.

Zabbix 7.x / 8.0 compatibility:

1. **Every API call failed on Zabbix 7.2 and later.** The auth token was sent as a
   JSON-RPC `auth` body property; 7.2 removed it and rejects unknown parameters.
   The provider now sends `Authorization: Bearer` from 6.4 and the body property
   below that — exactly one of the two, never both.
2. **Hosts and templates read back with no groups at all on Zabbix 7.2+.**
   `selectGroups` was replaced by `selectHostGroups`/`selectTemplateGroups`. `.get`
   methods ignore unknown parameters on *every* version, so this was a silent wrong
   answer rather than an error — the more dangerous of the two failure modes.
3. **Host proxy assignment was wrong on Zabbix 7.0+**: `proxy_hostid` became
   `proxyid` alongside a new `monitored_by`. Both models are handled; the resource
   attribute is unchanged.
4. **Every item update failed on Zabbix 7.x**: `hostid` became create-only at 7.0
   and `item.update` / `itemprototype.update` / `discoveryrule.update` reject it.
5. **HTTP `headers` and query fields on items, item prototypes and discovery rules**
   became an array of `{name, value}` at 7.0. Writes now match the server version;
   reads accept either shape.
6. **Preprocessing step 26 ("check for not supported value")** takes mandatory
   parameters from 7.0. An empty `params` is sent as `-1` (any error) and mapped
   back on read, so a config that omits it round-trips.
7. **Removed and read-only properties were sent on the item write path** and are
   hard errors from 7.0: `data_type` and `delta` (gone since Zabbix 3.4), plus the
   read-only `hosts` and `discoveryRule`. `discoveryRule` carried a `omitEmpty`
   struct tag, which `encoding/json` ignores, so it was serialised as `null` on
   every single call.
8. **The proxy data source was wrong on Zabbix 7.0+**: `proxy.get` dropped
   `selectInterface` and filtering on `host`.
9. **Templates could not be created on Zabbix 6.2+**: `template.create` was handed
   a host group id. This single mismatch accounted for every remaining acceptance
   failure on 7.0, 7.4 and 8.0.

Correctness:

10. **`zabbix_lld_dependent` could never be created.** The shared LLD schema
    defaulted `delay` to 3600; a dependent rule is driven by its master item and
    Zabbix requires 0.
11. **`zabbix_lld_trapper` failed on create** unless the user knew to write
    `delay = "0"`, for the same reason. `delay` is now pinned to `"0"` on both.
12. **No `zabbix_proto_item_*` resource could be updated on Zabbix 7.2+.**
    `itemprototype.update` was sent the create-only `ruleid`. Item prototypes were
    effectively write-once on both current Zabbix releases.
13. **Every HTTP item had a permanent, unappliable diff.** `post_type` defaulted to
    `"body"`, which is not one of its values (`raw`/`json`/`xml`) — it was copied
    from `retrieve_mode`, where `"body"` is valid. The default is now `"raw"`.
14. **The `zabbix_template` data source crashed the provider on every read.** The
    shared read path sets `templates`, which the data source schema did not declare,
    and `helper/schema` panics on an unknown address.
15. **`zabbix_graph` and `zabbix_proto_graph` never reached an empty plan on Zabbix
    8.0**, which returns a graph's items in a different order. Fixed by the
    `TypeList`→`TypeSet` conversion above, not by sorting the read result: a list
    must match the *config's* order, which need not agree with any field the server
    sorts by.
16. **Updating an LLD rule that already had a filter failed on Zabbix 7.2+.** The
    provider echoed back the formula ids the server had assigned, which 7.2+ rejects
    outright. They are now sent only for `evaltype = "custom"`, where the server
    requires them.
17. **`evaltype = "custom"` was unusable** — the condition `id` was not writable, so
    the call failed with "the parameter formulaid is missing".
18. **The computed `id` on every `macro` block was permanently empty.**
    `Macro.MacroID` was tagged `hostmacroids`, the `usermacro.get` *filter* name,
    rather than the object property `hostmacroid`.
19. **`templates_clear` could name an already-deleted template.** Terraform destroys
    a template in the same apply that removes it from a host or template, without
    ordering the unlink first; Zabbix 6.0 tolerated the stale id, 7.0+ makes it a
    hard error. Both the host and the template update paths now filter it against
    the ids the server still knows.
20. **Six collections could be added to but never emptied.** `omitempty` on a
    property Zabbix replaces wholesale means removing the last element sends no
    property at all, the server keeps what it had, and the next read puts it back:
    item `preprocessing` and `tag`, LLD `preprocessing`, trigger `dependencies` and
    `tag` on both triggers and trigger prototypes, and LLD `macro_path`. Zabbix
    6.0 additionally rejects `[]` on create where 7.0+ accepts it, so `macro_path`
    distinguishes "absent" from "empty" with a pointer.
21. **HTTP item `posts` and `proxy` could not be cleared**, for the same reason.
    They cannot simply lose `omitempty` — one `Item` struct serves all ten backend
    types and 7.0+ rejects properties that do not apply — so the four HTTP-only
    fields became pointers, where a pointer to `""` marshals and a nil one is
    omitted.
22. **A trigger `url` could not be removed once set**, for two stacked reasons: the
    validator rejected `""`, and the property was `omitempty`.
23. **All five data sources silently succeeded when their lookup matched nothing.**
    They ended in a read shared with the corresponding resource, where "found
    nothing" clears the id to report drift. Terraform accepted the placeholder
    `id-attribute-not-set` and the failure surfaced later and elsewhere as
    `Invalid parameter "/groupids/1": a number is expected` or, worse, "Database
    error occurred". Each data source now fails naming the lookup that missed.
24. Host macros and host/item tags could not be removed, and `zabbix_host` deletion
    sent properties the API rejects. (Fixed on the development branch after
    `v0.17.0` and unreleased until now.)
25. **Changing an item's `valuetype` to `text` or `log` failed on Zabbix 7.0 and
    later** whenever `trends` was not written out in the configuration. `trends` is
    computed from the value type, and the computed value survived a change of the
    value type it was derived from, so the item was updated with a trends period
    Zabbix refuses for those types (`Invalid parameter "/1/trends": value must be
    0`). It is now re-derived on every write. A text or log item that asks for a
    non-zero `trends` explicitly is refused with a message saying why, instead of
    failing that way on 7.0+ and sitting in a diff that never converged on 6.0.

Items 20 and 24 each group several instances of one root cause fixed together;
counted individually the total is higher. One root cause dominates: `omitempty`
on a property Zabbix reads as "leave as is" produced six separate bugs (the four
collections in 20, plus 21 and 22), and is written up in
[CONTRIBUTING.md](./CONTRIBUTING.md#the-omitempty-trap).

26. **A trigger expression containing a user macro never reached an empty plan.**
  The read path asked `trigger.get` for `expandExpression`, which turns the
  stored `{functionid}` tokens back into readable `/host/key` references — which
  the provider needs — but also expands user macros to their *values*, which it
  does not, and Zabbix offers no way to ask for one without the other. A
  configuration saying
  ```hcl
  expression = "min(/Internal HTTPS service/net.tcp.service.perf[https,,443],5m)>{$HTTPS.RESPONSE.SLOW}"
  ```
  was read back as `...>5`, so it never matched the configuration: every plan
  proposed rewriting the expression, applied, and proposed it again.
  The provider now reconstructs the expression the user wrote from the
  trigger's functions, items and hosts, which is what the Zabbix frontend does,
  and no longer asks for `expandExpression` at all. Whitespace, quoting, LLD
  macros and user macros — in the comparison and inside function parameters
  alike — all survive the round trip unchanged. It applies to `zabbix_trigger`
  and `zabbix_proto_trigger`, and to `recovery_expression` as well as
  `expression`.
  Nothing to change in your configuration. The first plan after upgrading is
  empty where it used to propose a rewrite.

27. **An item with `valuetype = "character"` and no `trends` could not be created
  on Zabbix 7.0 and later.** Trends are hourly minimum/average/maximum, so Zabbix
  keeps them for the two numeric value types only — but the provider's derivation
  treated *text and log* as the non-numeric pair and left `character` out, so a
  character item had `trends = "365d"` derived for it and the create was rejected
  with `Invalid parameter "/1/trends": value must be 0` naming an attribute the
  user had never written. Zabbix 6.0 accepted the same call and stored 0 anyway,
  so the configuration worked where it was written and broke on upgrade. The
  derivation now covers `character`, `log` and `text` alike.
28. **An item that stopped being `text` kept `trends = "0"` for ever.** The
  companion to 25, and the direction nothing had forced: with `trends` absent from
  the configuration the stored value survives a `valuetype` change, and moving
  *out* of a non-numeric type left the item with trends disabled and collecting
  none — silently, because state agreed with the server. `trends` is now derived
  in `CustomizeDiff`, so the value a silent configuration is going to get is in
  the **plan** rather than only in the applied result, on create and on a
  `valuetype` change that crosses the numeric boundary. A change *within* a class
  (unsigned to float, text to character) still leaves the stored value alone: a
  trends period set in the frontend and imported into a configuration that does
  not manage it belongs to the user.
29. **A user macro could not hold the empty string.** `value` carried a
   `StringIsNotWhiteSpace` validator, but Zabbix accepts and stores an empty
   macro value, and placeholder macros left empty are common in shipped
   templates. Beyond making `value = ""` unwritable, it made any host or
   template already carrying such a macro **impossible to import**: the read put
   `""` into state and no configuration could match it.
30. **A tag edit could be silently discarded.** `tagHash` hashed
   `key + "V" + value`, so `{key = "aV", value = "b"}` and
   `{key = "a", value = "Vb"}` hashed identically. `helper/schema`'s `diffSet`
   compares hash codes only and never inspects elements whose codes match, so an
   edit between two such tags produced no diff, no API call and no error.
31. **Two stray `fmt.Printf` calls** wrote to the plugin's stdout on every host
   update that touched a tag. The block they sat in was a no-op besides —
   `tagGenerate` never returns nil and `prepHosts` marshals a non-nil `Tags` to
   `[]` regardless — so it was removed entirely.
32. **`MacrosCreate` wrote the new macro id into `HostID`** instead of `MacroID`.
   Unreachable today, since macros are managed through `host.update` and
   `template.update`, but a landmine for whoever wires it up.
33. **An item could be managed by the wrong resource, and the first edit stopped it
  collecting data.** No `zabbix_item_*`, `zabbix_proto_item_*` or `zabbix_lld_*`
  resource compared the object's backend type against the type it represents, so
  `terraform import zabbix_item_agent.x <itemid-of-an-SNMP-item>` **succeeded**:
  `snmp_oid` is simply not an attribute of the agent resource, nothing in state
  recorded the type, and the next plan was empty. The item was then managed by the
  wrong resource, and the first change to any unrelated attribute sent the agent
  resource's own `type` along with it — Zabbix accepted the rewrite and the item
  silently stopped collecting. The same hole made a type changed in the frontend
  invisible: drift in the one property that decides what the object *is* was the
  one property never compared. A read now refuses an object whose type the resource
  does not represent, naming what it actually is and which resource takes it —
  `item 12345 is a SNMP agent item, not a Zabbix agent or Zabbix agent (active)
  item; import it as zabbix_item_snmp`. `zabbix_item_agent` and its two siblings
  accept both the passive and active agent types, which is the `active` attribute's
  whole job.
34. **`inventory_mode = "automatic"` destroyed the fields Zabbix populated itself.**
  Under automatic mode Zabbix fills inventory fields in from any item carrying an
  `inventory_link`. The read copied every field the server returned into state,
  including those, and the write then sent `""` for anything state held that the
  configuration no longer named — which is exactly that set. Verified end to end on
  8.0: a host with an `inventory` block naming `name`, the server having set `os`,
  planned `- os = "Linux 6.1 (auto-discovered)" -> null` and the apply **wiped it on
  the server**. Zabbix repopulated it and the next plan deleted it again: a permanent
  fight with the discovered data as the casualty. Under automatic mode the provider
  now sends exactly the fields the configuration names and reads back only those, so
  a field Zabbix owns is neither reported in state nor deleted. **Manual mode is
  unchanged** — there the configuration owns the whole inventory and deleting a line
  still clears the field. The cost is one deliberate exception to a deleted line
  reverting an attribute, and it is written on the attribute and in
  [MIGRATING.md §10](./MIGRATING.md#10-inventory-under-inventory_mode--automatic-leaves-unnamed-fields-alone):
  under automatic mode, write `""` to clear a field. Setting inventory fields under
  automatic mode is something Zabbix explicitly permits — probed, not assumed — so
  the provider does not invent a validation error for it.
35. **An `inventory` block whose fields were all empty could never reach an empty
  plan.** The read dropped an inventory object with nothing in it, leaving state at
  `inventory.# = 0` against a configuration holding one block, and no apply could
  close the gap. `inventory { location = "" }` on its own was enough to hit it on any
  version; it became ordinary under automatic mode, where clearing a field is written
  as `""`. A block the configuration holds is now reported back as a block. A host
  with no block at all still reads back as none, which is the case the old behaviour
  was protecting.

### Removed

- **`zabbix_application` and `data.zabbix_application`**, and the `applications`
  attribute on every item, item prototype and discovery rule. Applications were
  removed from Zabbix at 5.4; item tags replace them. See
  [MIGRATING.md §2](./MIGRATING.md#2-zabbix_application-is-gone--use-item-tags) and
  [§3](./MIGRATING.md#3-applications-is-gone-from-every-item).
- **`zabbix_item_aggregate` and `zabbix_proto_item_aggregate`.** Aggregate items
  were removed from Zabbix at 6.0; calculated items with aggregate functions replace
  them, with a translation table in
  [MIGRATING.md §4](./MIGRATING.md#4-aggregate-items-are-gone--use-calculated-items).
- **Legacy SNMP item attributes** — `snmp_version`, `snmp_community`,
  `snmp3_authpassphrase`, `snmp3_authprotocol`, `snmp3_contextname`,
  `snmp3_privpassphrase`, `snmp3_privprotocol`, `snmp3_securitylevel`,
  `snmp3_securityname`. Zabbix 5.0 collapsed the three SNMP item types into one and
  moved the credentials onto the host interface. Only `snmp_oid` remains on the item.
  See [MIGRATING.md §7](./MIGRATING.md#7-legacy-snmp-item-attributes-are-gone).
- All pre-6.0 code paths, including every version gate below 6.0 (net −1396 lines).
- `utils/template2terraform`, the standalone Python XML→HCL converter. It emitted
  `zabbix_application` and `zabbix_item_aggregate`, so its output no longer parses
  against this provider. It remains in history and on the frozen `master` branch.
- The root `example.tf` scratch file, superseded by `examples/`.

[Unreleased]: https://github.com/tpretz/terraform-provider-zabbix/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/tpretz/terraform-provider-zabbix/compare/v0.17.0...v2.0.0
