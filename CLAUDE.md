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

> Two known warts, tracked as work items: the seven `internal/zabbix/*_test.go` files call `log.Fatal` in `init()` when `TEST_ZABBIX_URL` is unset, so **a bare `go test ./...` hard-fails confusingly** — use `go test ./provider/`. They also had not compiled for years before the merge (a stale `NewAPI` call signature), because the nested `go.mod` hid them from `./...`. Library cruft (`.travis.yml`, `tests.sh`, `README.md`, the stale `UserAgent` string) came across with them.

## Build / test commands

```bash
go build ./...              # build the provider plugin
go vet ./provider/
go test ./provider/         # unit tests (schema validation) — no server needed
```

### Acceptance tests need live Zabbix (multi-version)

> **Being replaced.** The harness described below targets 4.0/5.0/5.4/6.0 — every one EOL but 6.0, and neither current Zabbix release is covered. PLAN.md Phase 1 replaces it with a standalone root-level `docker-compose.test.yml` for 6.0/7.0/7.4/8.0 that runs without the devcontainer. This is the *first* thing being built, so that everything after it is validated as it lands. Expect the commands below to change.

Acceptance tests talk to real Zabbix servers and mutate them. The `Makefile` drives the standard Terraform `TF_ACC` flow against four versions at once and is meant to run **inside the dev container** (`.devcontainer/`), which brings up `zabbix-web-40/50/54/60` alongside it:

```bash
make testacc                # runs test40 test50 test54 test60 (Zabbix 4.0 / 5.0 / 5.4 / 6.0)
make test40                 # single version; each sets ZABBIX_URL to the matching web container
```

The Makefile exports `TF_ACC=1`, `ZABBIX_USER=Admin`, `ZABBIX_PASS=zabbix` and points `ZABBIX_URL` at `http://zabbix-web-<ver>:8080/api_jsonrpc.php`; acc logs go to `provider/acc.log`. To run a single test against one server:

```bash
TF_ACC=1 ZABBIX_USER=Admin ZABBIX_PASS=zabbix \
  ZABBIX_URL=http://zabbix-web-40:8080/api_jsonrpc.php \
  go test -v -run TestAccResourceHost ./provider
```

Releases were cut by goreleaser on pushing a `v*` tag. **That trigger is removed on this branch** — see "GitHub Actions is disabled" above.

## Architecture

### Version-aware API client

`zabbix.NewAPI(Config)` (`internal/zabbix/base.go`) immediately calls `APIInfo.version` and stores the result in `Config.Version` as an **integer**: `major*10000 + minor*100 + patch` (e.g. 6.0.13 → 60013, 7.4 → 70400). Behaviour across the codebase branches on this number rather than a version string, because the provider must support a range of Zabbix versions. When something changed between Zabbix versions, gate it with `api.Config.Version >= NNNNN` and keep the old path.

With the 6.0 floor, the gates that **survive** are:

| Gate | What it covers |
|---|---|
| `>= 60200` | template groups split from host groups |
| `>= 60400` | bearer auth; template `vendor_name`/`vendor_version` |
| `>= 70000` | proxy model rewrite, `monitored_by`, LLD header arrays, browser items |
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

**To add a new item backend type**: create `resource_<type>_common.go` following an existing one (snmp is the fullest example), then register the three resource constructors in `provider/provider.go`. Seven types are still missing — db_monitor, ipmi, ssh, telnet, jmx, script, browser — and are enumerated in PLAN.md § 4a. Follow the S1–S8 definition of done in PLAN.md § "The unit of work": every new resource lands with an acceptance test including an import step, a sweeper, docs and an example.

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

### utils/template2terraform

`utils/template2terraform` is a standalone **Python 3** script that converts a Zabbix template XML export into equivalent Terraform configuration — a code-generation helper, not part of the Go build. It is currently undocumented and untested; PLAN.md Phase 5 forces a keep-or-move decision on it.

## Testing expectations

`testAccPreCheck` (`provider/provider_test.go`) requires `ZABBIX_URL`, `ZABBIX_USER`, `ZABBIX_PASS`.

Current acceptance coverage is thinner than the file listing suggests — nine test files contain only `package provider`, no data source or item-prototype resource is tested, and although every resource declares `ImportStatePassthrough`, no test exercises import. See API-COVERAGE.md § 4 for the itemised gaps. When touching a resource, add the missing test rather than leaving the stub.

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
