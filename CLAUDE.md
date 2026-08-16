# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

This is the **terraform-provider-zabbix** repo. Active development is on the **`v2`** branch.

## Read first: PLAN.md

**[PLAN.md](./PLAN.md) and [API-COVERAGE.md](./API-COVERAGE.md) are the source of truth for what this project is doing and why.** Read PLAN.md before making non-trivial changes — much of what this file describes below is the *current* state, and a good deal of it is scheduled for deliberate removal.

Summary of the decisions recorded there:

| | |
|---|---|
| Branch | All work on **`v2`**, cut from `testenv`. `master` and `testenv` are **frozen** — no commits, no backports, no re-tagging. The published `v0.x` releases stay as they are. |
| Next release | **`v2.0.0`** (v1.x deliberately skipped), cut only when phases 0–3 are complete. |
| Zabbix floor | **6.0 LTS.** 4.0, 5.0 and 5.4 support is being *deleted*, not merely untested. |
| Test matrix | 6.0, 7.0, 7.4 release-gating; 8.0 via `ubuntu-trunk` non-blocking. |
| API client | Merged into this repo as `internal/zabbix`; the submodule is retired. |

### Current state — measured, not guessed

**The matrix is green on all four versions** — 106 tests, 86 of them acceptance, roughly
205-215s per version:

| Version | Result |
|---|---|
| 6.0.48 | pass (3 skips — templategroup tests, gated to 6.2+) |
| 7.0.29 | pass (1 skip) |
| 7.4.13 | pass (1 skip) |
| 8.0-trunk | pass (1 skip) |

Getting here took Phases 0-2a plus 3a. Findings from that work that contradict what was previously assumed, all verified empirically against live servers:

- **Strict parameter validation arrived in 7.0, not 7.2, and is per-method.** `item.create`, `itemprototype.create` and `discoveryrule.create` reject unknown object properties from 7.0. `host.create` and `graph.create` are still lenient even on 8.0. `.get` methods silently ignore unknown params on *all* versions — so a stale `selectGroups` on 7.2+ is a **silent wrong answer**, not an error, which is far more dangerous than a hard failure.
- **`hostid` is create-only from 7.0.** `item.update`, `itemprototype.update` and `discoveryrule.update` reject it, so every item update on 7.x would have failed. Handled by `prepItemsUpdate`/`prepLLDsUpdate`.
- **`selectGroups` is a working alias for template groups on 6.2-7.1**, which is why the template read path can use it below `V72`.
- **`templates_clear` could name an already-deleted template.** Terraform destroys a template in the same apply that drops it from a host's `templates`, without ordering the host update first. 6.0 tolerated the stale id; 7.0+ makes it a hard error. `resource_host.go` now filters `templates_clear` to ids the server still knows.

### Test fixture idiom you must know about

Acceptance fixtures are written against `zabbix_templategroup` and **textually rewritten** to `zabbix_hostgroup` by `hcl(t, ...)` in `provider_test.go` when the server is below 6.2. It is a plain string swap, so any config needing *both* group types must use distinct labels and group names — the shared fixtures use `testtmplgrp`/`test-template-group` against `testgrp`/`test-group`. Get this wrong and the pre-6.2 run silently collapses both into one resource.

### GitHub Actions is disabled

Actions is switched off repository-wide, **and** `.github/workflows/release.yml` is separately neutered on this branch (manual dispatch only, typed confirmation required, plus a guard refusing to run from `v2`). Nothing on this branch should execute in CI until the v2 line is ready to release. Do not restore the tag trigger without reading PLAN.md § "Branch and release strategy" — an unscoped `v*` tag trigger would let this branch publish over the `v0.x` release line.

## The API client: internal/zabbix

The provider's Zabbix API client lives at **`internal/zabbix`**. It used to be a git submodule (`github.com/tpretz/go-zabbix-api`) wired in with a `replace` directive; that was retired in Phase 0. There is no submodule, no `.gitmodules`, and no `replace` — `git submodule status` returns nothing. The client's full history was rewritten under `internal/zabbix/` before merging, so `git blame` and `git log --follow` work on it.

Because it is an `internal/` package it cannot be imported outside this module, which is intended — the provider is its only consumer. Edit it directly; there is nothing to re-tag.

> The package has no tests of its own. Its original `TEST_ZABBIX_*` harness was deleted in v2 — it had not compiled since 2023, hidden from `./...` by the nested `go.mod`, and duplicated coverage the acceptance suite provides against four live servers. `go test ./...` is clean; the caveat about it hard-failing no longer applies.

## Build / test commands

```bash
go build ./...              # build the provider plugin
go vet ./provider/
go test ./provider/         # unit tests (schema validation) — no server needed
```

### Acceptance tests need live Zabbix (multi-version)

`docker-compose.test.yml` in the repo root brings up four independent stacks — **6.0, 7.0,
7.4 and 8.0** — each with its own PostgreSQL, on localhost ports 8060/8070/8074/8080. Plain
`docker compose`; no devcontainer needed. Full bringup is about 12 seconds.

```bash
make testenv-up            # all four; testenv-up-74 for one
make testacc               # 6.0, 7.0, 7.4 - the release-gating set
make testall               # adds 8.0, which is non-blocking
make test-one TEST=TestAccResourceHost VER=74
make testenv-down
```

8.0 tracks the `ubuntu-trunk` nightly, so it is a moving target and never gates a release.
Adding a version is a `VERSIONS` entry in the `Makefile` plus a three-line compose block.

Health checks poll `apiinfo.version` rather than the web root: the frontend answers HTTP
before the database schema is loaded, which is the classic flaky-start cause.

**If many tests fail in under a second with `already exists`,** that is leftover state from
an aborted run, not your code. Clear it:

```bash
TF_ACC=1 ZABBIX_USER=Admin ZABBIX_PASS=zabbix \
  ZABBIX_URL=http://localhost:8074/api_jsonrpc.php \
  go test ./provider/ -sweep=all
```

The suite assumes **exclusive use of a stack**. Two processes against the same version —
two agents, or a stray `make test-one` during a full run — collide on fixtures, because
fixture names are deliberately stable (`test-group`, `testhost`, …) so the sweepers can
find them.

Full details, including the current results table, are in [TESTING.md](./TESTING.md).

## Architecture

### Version-aware API client

`zabbix.NewAPI(Config)` (`internal/zabbix/base.go`) immediately calls `APIInfo.version` and stores the result in `Config.Version` as an **integer**: `major*10000 + minor*100 + patch` (e.g. 6.0.13 → 60013, 7.4 → 70400). Behaviour across the codebase branches on this number rather than a version string, because the provider must support a range of Zabbix versions. When something changed between Zabbix versions, gate it with `api.Config.Version >= NNNNN` and keep the old path.

With the 6.0 floor, the gates that **survive** are:

| Gate | What it covers |
|---|---|
| `>= 60200` | template groups split from host groups |
| `>= 60400` | bearer auth; template `vendor_name`/`vendor_version`; `snmp_walk_value` and `snmp_walk_to_json` preprocessing |
| `>= 70000` | proxy model rewrite, `monitored_by`, LLD header arrays, browser items; `snmp_get_value` preprocessing, and `matches_regex` on discovery rules |
| `>= 70200` | `selectHostGroups`/`selectTemplateGroups` replacing `selectGroups` |
| `>= 70400` | LLD rule prototypes |

Everything gated below `60000` is dead and is being deleted (Phase 2b) — the legacy SNMP v1/v2/v3 item types, applications, aggregate items, the v4 inventory/tag fallbacks, and roughly 30 comparison sites across provider and tests. **Do not add new gates below `60200`.** New gates should use named constants (`zabbix.V62`, `V64`, `V70`, `V72`, `V74`) rather than bare integers.

`API.CallWithError` / `CallWithErrorParse` are the low-level request methods; the library's resource files (`host.go`, `item.go`, `trigger.go`, `template.go`, `lld.go`, `graph.go`, …) wrap specific `*.create/get/update/delete` calls and define structs with Zabbix's JSON field names.

### Provider auth

`providerConfigure` (`provider/provider.go`) accepts **either** username+password **or** a `token`; all three are optional but one path is required (it errors "credentials required" otherwise). Env fallbacks: `ZABBIX_USER`/`ZABBIX_USERNAME`, `ZABBIX_PASS`/`ZABBIX_PASSWORD`, `ZABBIX_TOKEN`, and `ZABBIX_URL`/`ZABBIX_SERVER_URL`. With a token, `api.Auth` is set directly and `Login()` is skipped. The constructed `*zabbix.API` is the `meta interface{}` passed to every CRUD func.

### Provider: the item / prototype / LLD triad

The bulk of the provider is item definitions, and this is the pattern to understand before touching them. Each item *backend type* (snmp, agent, http, trapper, simple, external, internal, snmptrap, calculated, dependent — plus `aggregate`, which is being removed) is exposed as up to **three Terraform resources** built from one `resource_<type>_common.go` file:

- `zabbix_item_<type>` — a normal item
- `zabbix_proto_item_<type>` — an item **prototype** (belongs to an LLD rule)
- `zabbix_lld_<type>` — an LLD discovery rule

Each `resource_<type>_common.go` provides two callbacks and wires them into all three resources:

- a **mod func** `itemXxxModFunc(d, m, *zabbix.Item)` — Terraform state → Zabbix struct
- a **read func** `itemXxxReadFunc(d, m, *zabbix.Item)` — Zabbix struct → Terraform state
  (LLD variants use `*zabbix.LLDRule` with their own `lldXxxModFunc`/`lldXxxReadFunc`.)

`common_item.go` supplies the shared machinery: `itemGetCreateWrapper` / `protoItemGetCreateWrapper` / `lldGetCreateWrapper` (and Read/Update equivalents) are factories that take the mod+read funcs and return Terraform CRUD closures; `buildItemObject` handles common fields; `resourceItemCreate/Read/Update` plus a `prototype bool` flag route to the right API method. Schemas are assembled with `mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema, …, typeSpecificSchema)` (`utils.go`). `common_lld.go`, `common_macro.go`, and `common_tag.go` play the same role for LLD rules, macros, and tags.

**To add a new item backend type**: create `resource_<type>_common.go` following an existing one (snmp is the fullest example), then register the three resource constructors in `provider/provider.go`. Seven types are still missing — db_monitor, ipmi, ssh, telnet, jmx, script, browser — and are enumerated in PLAN.md § 4a. Follow the S1–S9 definition of done in PLAN.md § "The unit of work": every new resource lands with an acceptance test including an import step, a sweeper, docs and an example.

### Collections: do not model unordered server data as an ordered list

**A `TypeList` asserts that order is semantic.** Use it only when the Zabbix API
guarantees an order *and* that order carries meaning. Otherwise use `TypeSet` — the
server's return order is an implementation detail and is subject to change between
versions, which is exactly how the 8.0 graph regression happened.

Do not "fix" an ordering mismatch by sorting the read result. Sorting picks some order,
but a `TypeList` must match the *config's* order, which need not agree with any field
the server sorts by. Sorting graph items by `sortorder` fixed nothing and broke 7.4.

| Collection | Type | Why |
|---|---|---|
| item / LLD `preprocessing` | `TypeList` | steps execute in sequence — order is genuinely semantic |
| host `inventory` | `TypeList` | a single nested block, not a collection |
| graph `item`, host `interface`, LLD filter `condition` | `TypeSet` | converted; server return order is not stable across versions |
| trigger `dependencies`, tags, macros | `TypeSet` | |

LLD conditions were confirmed unordered empirically, not by inspection: submitting
`{#CCC},{#AAA},{#BBB}` comes back in submission order on 6.0 but sorted by formulaid on
7.4/8.0. The order is not merely undefined — it *changed between versions*.

### A set's hash must cover every user-settable attribute

This is the trap, and it is worse than the ordering problem it replaces.
`helper/schema`'s `diffSet` short-circuits on `reflect.DeepEqual(os.listCode(), ns.listCode())`
(`schema.go:1609`) — it compares **hash codes only** and never looks at elements when they
match. An attribute left out of the hash therefore cannot be seen to change, and the edit
is **silently discarded**: no diff, no API call, no error.

Hash every user-settable field and exclude only server-assigned ids (which config lacks,
and which would otherwise replace every element on every plan). `hashElementExcept` in
`provider/utils.go` does this generically so a newly added field cannot be forgotten.

Because an edited element arrives with no id, `hostReuseInterfaceIDs` in
`resource_host.go` reassigns the prior `interfaceid` of the same type. Without it Zabbix
is asked to delete and recreate an interface, which it refuses once items are bound to it.

**Sets cannot be indexed from HCL.** `zabbix_host.x.interface[0].id` no longer parses;
use `one(...)`. This is the change most likely to break an existing config.

### Shared schema helpers & the lookup-table idiom

- `common_tag.go` — shared `tagSetSchema` (a `TypeSet` of key/value) plus `tagGenerate`/`flattenTags`, reused by host, trigger, and item resources.
- `common_macro.go` — `macroSetSchema` (a `TypeSet`) plus `macroGenerate`/`flattenMacros`, reused by host and template resources.

Enum-like fields map between human-friendly Terraform strings and Zabbix's numeric codes using a forward map plus a generated reverse map and value-list, e.g. in `resource_snmp_common.go`:

```go
var SNMP_LOOKUP = map[string]zabbix.ItemType{...}
var SNMP_LOOKUP_REV = map[zabbix.ItemType]string{}   // filled by an init-style anon func
var SNMP_LOOKUP_ARR = []string{}                     // used for validation.StringInSlice + docs
```

The `_REV`/`_ARR` are populated by a package-level `var _ = func() bool { ... }()` block. Follow this idiom for new enum fields so validation messages and reverse lookups stay in sync.

### Provider wiring & logging

`provider/provider.go` is the single registry of all data sources and resources. Logging goes through the package-level `log` (`provider/log.go`), emitting `[TRACE]`/`[DEBUG]`/… lines; enable with `TF_LOG=TRACE`.

### Documentation is generated — never hand-edit `docs/`

`docs/` is produced by [`terraform-plugin-docs`](https://github.com/hashicorp/terraform-plugin-docs), pinned at **v0.25.0** in the `Makefile` and run through `go run <module>@<version>` (so it is not in `go.mod`). An edit made directly to a file under `docs/` is thrown away by the next generation.

```bash
make docs           # regenerate docs/
make docs-check     # regenerate and fail if docs/ moved -- the CI gate
make docs-validate  # check docs/ against the Terraform Registry's layout rules
```

Three inputs feed a page, and which one to reach for is the thing to get right:

| Input | Becomes |
|---|---|
| the schema `Description` on every attribute, and on the `schema.Resource` itself | the page intro and the argument reference |
| `templates/resources/<name>.md.tmpl` (no `zabbix_` prefix) | hand-written prose around it; falls back to `templates/resources.md.tmpl` |
| `examples/resources/<full_name>/resource.tf` (with the prefix) | the "Example Usage" section |
| `examples/resources/<full_name>/import.sh` | the "Import" section |

Data sources use `templates/data-sources/` and `examples/data-sources/<full_name>/data-source.tf`.

**A weak generated page almost always means a weak schema `Description`, not a generation problem.** Fix it in the schema, where it also reaches `terraform providers schema -json` and editor tooling, rather than by writing the page by hand. Two tests hold the line: `TestSchemaDescriptionsPresent` requires a `Description` on every attribute, and `TestEnumDescriptionsListValues` requires an enum's description to list every value its validator accepts — build that list from the `_LOOKUP_ARR` so it cannot drift.

Every `_ARR` is sorted at init (`TestEnumValueListsAreSorted`). This is not cosmetic: they are built by ranging over a map, Go randomises map iteration order, and an unsorted list makes `make docs-check` fail at random. `TRIGGER_PRIORITY_ARR` is ordered by severity code instead, deliberately.

> `utils/template2terraform`, the standalone Python XML→HCL converter, was **deleted** in Phase 5. It emitted `zabbix_application` and `zabbix_item_aggregate`, so its output no longer parses against this provider. It is in history if anyone wants to revive it — in its own repository.

## Testing expectations

`testAccPreCheck` (`provider/provider_test.go`) requires `ZABBIX_URL`, `ZABBIX_USER`, `ZABBIX_PASS`.

Coverage is now **106 tests, 86 of them acceptance**, green on all four versions at roughly
205–215s each. Every registered resource and data source has a test with an import step and
a `CheckDestroy`; there is one `ImportStateVerifyIgnore` suite-wide (proxy PSK, which
`proxy.get` never returns). Drift, negative paths, `ForceNew`, provider configuration and
scalar boundaries are covered — see PLAN.md § Phase 8.

**Do not treat that as licence to add a resource without tests.** Of the 24 defects fixed in
v2, most were found by a test written where coverage was absent, and every one had been
invisible for years. `S1`–`S9` and `C1`–`C7` below are the bar.

### `S9`: a resource is not tested until it has been built from its minimum, and taken back

This is the mirror of PLAN.md § "The unit of work"; PLAN.md remains the source of
truth. Two questions, neither of which any other criterion asks:

> **Does the resource work when the user sets only what is required?**
> **Can what was set be unset?**

Six defects came from those two, all of them invisible to `C1`–`C7` and `E1`–`E6`,
because every fixture in the suite was written by somebody who already knew the
answer and set the attribute: `zabbix_lld_trapper` uncreatable without an explicit
`delay = "0"`; a `preprocessor` block failing on create when `error_handler` was
omitted; `zabbix_host` refusing a host with no interface; HTTP `posts`/`proxy` and
trigger `url` settable but not clearable; HTTP `status_codes`/`timeout` the same;
the four SNMPv3 credentials the same, and forbidden by a `ValidateFunc` besides.

| | Case | What it must do |
|---|---|---|
| S9a | **the documented minimum** | only the `Required` attributes — what the generated docs tell a user to write — applies and plans clean. Needing more is a finding: either the attribute should be `Required`, or it needs a working default |
| S9b | **each optional block at its own minimum** | S9a omits optional blocks entirely, so it never reaches the defaults declared *inside* one — where the `error_handler` defect lived. Each block gets its own minimum, carrying only what the block marks `Required` |
| S9c | **every `Default:` against a live server** | the empty plan after S9a/S9b is the check. `""` is legal for some Zabbix properties and rejected outright for others, so probe — do not reason from the documentation |
| S9d | **set, then unset** | every optional scalar that can hold an empty value is set to a non-default and then returned to empty, and the plan after the clear must be empty |
| S9e | **the clear reaching the server** | where the property is *merged* rather than replaced — host `inventory`, interface `details` — assert against a re-read, as `C6` does. State is written by the provider's own read, so a dropped clear still looks right there |

`S9d` is the scalar twin of `C6`. The usual mechanical cause is an `omitempty`
struct tag on a property the user is allowed to empty (see CONTRIBUTING.md
§ "The `omitempty` trap"), but it is not the only one: `d.GetOk` reports `""` as
"not set" and drops the key just as thoroughly — that is how the host inventory
fields were unclearable while every struct tag involved was correct.

Two traps, both of which hid a defect behind something that looked fine:

- **A default hides the empty value.** An attribute with no default reaches `""` by
  being omitted, so a fixture may stumble over it. An attribute *with* a default
  only reaches `""` if the user writes it out, and no fixture ever does.
- **A `ValidateFunc` can make a legal value unreachable.** `StringIsNotWhiteSpace`
  on an optional attribute forbids the empty string outright. Check the server
  agrees before adding one.

The tests are `provider/acc_minimal_test.go` (S9a–S9c) and
`provider/acc_clearable_test.go` (S9d–S9e).

### `U1`–`U4`: an attribute is not covered until it has been changed in life

This is the mirror of PLAN.md § "The unit of work"; PLAN.md remains the source of
truth.

Creating a resource and destroying it exercises two thirds of what a user does. The
other third — editing a value on something that already exists — is the most common
operation of the three, and until `U1`–`U4` nothing checked it systematically. Two
failure modes, both of which shipped in this codebase:

| | Case | What it must do |
|---|---|---|
| U1 | **changed in life** | every settable attribute is changed on an existing resource, and the new value asserted against a **server re-read**, not merely Terraform state |
| U2 | **and it was an update** | a `plancheck.ExpectResourceAction(...Update)` on the step. Without it a wrongly-`ForceNew` attribute passes: Terraform destroys and recreates, the end state matches, the test is green, and the user has silently lost their item's history |
| U3 | **create-only is `ForceNew`** | every attribute Zabbix refuses to update is `ForceNew` and asserts replacement |
| U4 | **nothing else is `ForceNew`** | no attribute is `ForceNew` that Zabbix would have accepted an update for — probed against live servers, never inferred |

`U3` is how `hostid` (create-only from 7.0) and prototype `ruleid` (create-only from
7.2) should have been found; both were instead found by accident, and `ruleid` had
made **every** `zabbix_proto_item_*` resource un-updatable on current Zabbix.

`U4` matters more than it looks. Replacing a Zabbix item **discards its history**, so a
`ForceNew` that need not be there is silent data loss rather than an inconvenience.
Probing found all three existing flags correct — and prototype `ruleid` correct on 6.0
for the worse of the two possible reasons: `itemprototype.update` *accepts* it there and
silently ignores it, so without `ForceNew` Terraform would report a move that never
happened.

The lists in `provider/acc_update_test.go` are the enforcement: the coverage map, the
`ForceNew` set and the exemptions are each checked against the live schema, so an
attribute added without update coverage fails the build. **Exempt by name with a
reason, never by omission** — an attribute that quietly falls out of a list is exactly
how `ruleid` stayed unnoticed.

### `R1`–`R2`: an attribute is not covered until the line has been deleted

This is the mirror of PLAN.md § "The unit of work"; PLAN.md remains the source of
truth.

`S9d` sets an attribute to an *empty value*. `R1`–`R2` **delete the line**, which is
the edit a user actually makes after changing their mind. What that means turns
entirely on one schema flag:

| | Case | What it must do |
|---|---|---|
| R1 | **`Optional` with a `Default:`** | deleting the line plans a revert to the default and the provider must **send** it, asserted against a server re-read. The failure mode is the `omitempty` one: a plan showing the revert while the server keeps the old value, invisible in state because the provider's own read wrote it |
| R2 | **`Optional + Computed`** | deleting the line produces **no diff at all** and the value sticks for ever. Whether that is right is a decision, and the decision — with what derives the value and what the user writes to get back — is the deliverable |

69 declarations carry a `Default:` and 5 are `Optional + Computed`; both sets are
enumerated from `Provider().ResourcesMap` and grouped by pointer identity, exactly as
`U1`–`U4` are, so an attribute given either flag tomorrow fails
`TestRemovalCoverageComplete` until it is covered or exempted **by name with a reason**.
The guard also checks the *class* in both directions: adding a `Default:` to an
`Optional + Computed` attribute, or dropping one, moves it between the two halves of
the criterion and must not do so silently.

R1 came back clean — every default reaches the server on all four versions — with two
findings worth remembering:

- **A default can be unreachable.** Item and LLD `interfaceid` default to `"0"`, which
  means *no interface* and which no server accepts for an object on a host that has
  interfaces (omitting the property fails too). Assert the failure rather than
  exempting.
- **The default and the stored value need not agree.** `zabbix_proxy` `address`/`port`
  revert to *empty* server-side; the provider reports the defaults back so an active
  proxy has no permanent diff. Assert the server against the server, the default
  against state.

All five R2 attributes were judged **intended** — `zabbix_host.name` and
`zabbix_template.name` (Zabbix derives the visible name from `host`), `interface.port`
(per-type default, and the one that *does* revert, because `hostInterfaceHash`
normalises an absent port before hashing), item `trends` (derived from `valuetype`) and
trigger `correlation_mode` (a `Default:` would break configurations predating it). None
was converted. Making `name` revert would need a `CustomizeDiff` re-derivation, which
would clobber the visible name of every host imported into a configuration that does
not manage it — worse than the trap it removes.

The tests are `provider/acc_removal_test.go` and `provider/acc_removal_host_test.go`.

### `C1`–`C7`: a collection attribute is not tested until it is tested plural

This is the mirror of PLAN.md § "The unit of work"; PLAN.md remains the source of
truth. It is repeated here because the rules are only useful if they are read
*before* the test is written.

Every collection bug found in this project hid behind a fixture that used exactly
**one element**: the 8.0 graph reordering, the LLD formula ids the provider echoed
back to a server that rejects them, `evaltype = "custom"` being unusable, and a
family of collections that could be added to but never emptied (`preprocessing`,
item and trigger `tags`, trigger `dependencies` — all `omitempty` on a property
Zabbix replaces wholesale). One element cannot distinguish a set from a list,
cannot show an ordering assumption, and cannot show an identity assumption.

| | Case | What it must do |
|---|---|---|
| C1 | **none** | attribute omitted entirely (where it is optional) — creates, plans clean, and imports back empty |
| C2 | **one** | the trivial case |
| C3 | **many** | three elements where the server may reorder, two otherwise. At least two must be *of the same kind*, so element identity is proven to come from content and not from position |
| C4 | **reordered** | the same elements in a different order, as a `PlanOnly: true` step. A **set** must plan empty; a **list** must instead produce a diff, and the new order must survive the round trip — that is what claiming `TypeList` means |
| C5 | **edit one of many** | change one attribute of one element and assert the *others* are untouched — and, where the object has a server-assigned id, that the untouched elements kept theirs |
| C6 | **remove one, then all** | N → N-1 → 0, and the removal must be shown **reaching the server**, not merely leaving state |
| C7 | **import at full size** | `ImportStateVerify` with the collection at its largest. The only check that the flatten function and the set hash agree |

Two rules that follow from this:

- **Assert set elements by content** — `TestCheckTypeSetElemNestedAttrs`,
  `TestCheckTypeSetElemAttrPair` — never by index. A set's indices in test state
  are positional artefacts of the JSON-state shim and mean nothing.
- **C6 needs a server-side check.** Zabbix's update calls replace collections
  wholesale, so an omitted element is a deletion — but only if the provider sends
  the property at all. A collection the provider silently drops still looks right
  in state, because state is written by the provider's own read. The
  `testAccCheck*Count` helpers in `provider/acc_collection_test.go` re-read the
  object from Zabbix and count what actually came back; use them for every C6 step.

**Shared machinery is tested plural once, not eleven times.** `tag`, `macro`,
`preprocessor` and the item/LLD headers each come from one file and are merged
into many resources, so full `C1`–`C7` coverage lives in one place per code path
and the other resources keep a single-element smoke check. Where that decision was
made is written next to the collection; the index is in the header comment of
`provider/acc_collection_test.go`. Note that item and LLD preprocessing are *not*
the same code path — `common_item.go` and `common_lld.go` each have their own — so
each has its own test.
