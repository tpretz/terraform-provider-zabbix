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
| API client | `go-zabbix-api` is being merged into this repo as `internal/zabbix`; the submodule is retired. |

### Current state — things that are known broken

The provider **cannot talk to a Zabbix 7.2+ server at all**. Do not be surprised by this; it is the main thing v2 exists to fix (PLAN.md Phase 2a):

- the auth token is sent as a JSON-RPC `auth` body property (`go-zabbix-api/base.go:26`) — removed upstream in 7.2 in favour of an `Authorization: Bearer` header (available from 6.4)
- `selectGroups` (`provider/resource_host.go:596,625`, `provider/resource_template.go:126,153`) — removed in 7.2
- `selectApplications` (`provider/common_item.go:275`) — applications were removed in 5.4
- `proxy_hostid` (`go-zabbix-api/host.go:63`) — renamed `proxyid` plus `monitored_by` in 7.0

7.2 also made unknown request parameters a hard error rather than silently ignoring them, so any stale key now fails the call outright.

### GitHub Actions is disabled

Actions is switched off repository-wide, **and** `.github/workflows/release.yml` is separately neutered on this branch (manual dispatch only, typed confirmation required, plus a guard refusing to run from `v2`). Nothing on this branch should execute in CI until the v2 line is ready to release. Do not restore the tag trigger without reading PLAN.md § "Branch and release strategy" — an unscoped `v*` tag trigger would let this branch publish over the `v0.x` release line.

## The go-zabbix-api submodule

The provider's Zabbix API client, `github.com/tpretz/go-zabbix-api`, is vendored here as a **git submodule** at `./go-zabbix-api`, and `go.mod` points the module at it:

```
replace github.com/tpretz/go-zabbix-api => ./go-zabbix-api
```

So the two codebases are developed together in this one repo. After a fresh clone (or when the submodule shows a leading `-` in `git submodule status`):

```bash
git submodule update --init --recursive
```

The submodule uses an **SSH** URL (`git@github.com:tpretz/go-zabbix-api.git`). Editing files under `go-zabbix-api/` changes the API client the provider builds against immediately — no re-tag needed — but those are commits in the *submodule's* repo, and the parent repo separately records which submodule commit is pinned.

> **Scheduled for removal.** PLAN.md Phase 0 merges this submodule into the repo as `internal/zabbix` and drops the `replace` directive — the two codebases already move together, and the split costs a `sed -i '/^replace/d' go.mod` hack at release time plus SSH access to a second repo for contributors. Write new client code expecting that destination.
>
> The v4-legacy inventory-mode/tag handling in `go-zabbix-api/host.go` that this branch previously preserved is **no longer wanted** — the 6.0 floor removes the 4.0 test target that justified it. It comes out in Phase 2b along with the legacy SNMP v1/v2/v3 item model and the applications code paths.

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

`zabbix.NewAPI(Config)` (`go-zabbix-api/base.go`) immediately calls `APIInfo.version` and stores the result in `Config.Version` as an **integer**: `major*10000 + minor*100 + patch` (e.g. 6.0.13 → 60013, 7.4 → 70400). Behaviour across the codebase branches on this number rather than a version string, because the provider must support a range of Zabbix versions. When something changed between Zabbix versions, gate it with `api.Config.Version >= NNNNN` and keep the old path.

With the 6.0 floor, the gates that **survive** are:

| Gate | What it covers |
|---|---|
| `>= 62000` | template groups split from host groups |
| `>= 64000` | bearer auth; template `vendor_name`/`vendor_version` |
| `>= 70000` | proxy model rewrite, `monitored_by`, LLD header arrays, browser items |
| `>= 72000` | `selectHostGroups`/`selectTemplateGroups` replacing `selectGroups` |
| `>= 74000` | LLD rule prototypes |

Everything gated below `60000` is dead and is being deleted (Phase 2b) — the legacy SNMP v1/v2/v3 item types, applications, aggregate items, the v4 inventory/tag fallbacks, and roughly 30 comparison sites across provider and tests. **Do not add new gates below `62000`.** New gates should use named constants (`zabbix.V62`, `V64`, `V70`, `V72`, `V74`) rather than bare integers.

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
