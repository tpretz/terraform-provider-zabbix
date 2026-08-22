# Development

Everything you need to build, test, document and extend the provider.

Companion documents: [CONTRIBUTING.md](./CONTRIBUTING.md) for the patterns and
traps a change has to respect, [TESTING.md](./TESTING.md) for the live
multi-version acceptance harness, [MAINTAINING.md](./MAINTAINING.md) and
[RELEASING.md](./RELEASING.md) for the maintenance and release runbooks,
[PLAN.md](./PLAN.md) for what the project is doing and why,
[API-COVERAGE.md](./API-COVERAGE.md) for the gap checklist.

## Requirements

`.tool-versions` pins the toolchain:

| | |
|---|---|
| Go | 1.25.12 |
| Terraform | 1.8.5 — the acceptance harness shells out to it |
| Docker | Compose v2, for the Zabbix test stacks |
| Python 3 | not required; nothing in the build or test path uses it |

## Build and unit tests

```bash
make build      # go build ./...
make vet        # go vet ./...
make test       # go test ./provider/ -- schema and state-migration unit tests, no server
gofmt -l .      # must print nothing; CI fails on any output
make            # lists every target
```

`make test` is fast and needs nothing running. It covers the schema
consistency checks (`provider/schema_enum_test.go`) and the state upgraders
against checked-in `v0.17.0` fixtures.

## Acceptance tests

Acceptance tests talk to real Zabbix servers and mutate them. Four stacks —
6.0, 7.0, 7.4 and 8.0-trunk — come up with plain `docker compose`:

```bash
make testenv-up                                 # all four, waits until the APIs answer
make testacc                                    # 6.0 + 7.0 + 7.4, the release gate
make test-one TEST=TestAccResourceHost VER=74   # day-to-day iteration
```

**[TESTING.md](./TESTING.md) is the full guide** — single-version stacks,
per-version logs, the 8.0 caveat, adding a Zabbix version, and troubleshooting.

## Repository layout

| Path | |
|---|---|
| `main.go` | plugin entrypoint |
| `provider/` | the provider: one file per resource family, plus the shared machinery |
| `internal/zabbix/` | the Zabbix JSON-RPC API client |
| `docs/` | **generated** registry documentation — do not hand-edit |
| `templates/` | hand-written prose that `tfplugindocs` renders into `docs/` |
| `examples/` | runnable HCL, pulled into the generated pages |
| `docker-compose.test.yml` | the four Zabbix test stacks |

### The API client is in-tree

`internal/zabbix` was a git submodule (`github.com/tpretz/go-zabbix-api`) until
Phase 0. It is not any more: there is no `.gitmodules`, no `replace` directive,
and `git submodule status` returns nothing. Edit it directly — there is nothing
to re-tag. Being an `internal/` package it cannot be imported outside this
module, which is intended; the provider is its only consumer.

## Documentation

`docs/` is **generated** by
[`terraform-plugin-docs`](https://github.com/hashicorp/terraform-plugin-docs).
Editing a file under `docs/` by hand loses the edit on the next generation.

```bash
make docs           # regenerate docs/ from the schema, templates/ and examples/
make docs-check     # regenerate and fail if docs/ moved — run this before pushing
make docs-validate  # check docs/ against the Terraform Registry's layout rules
```

`tfplugindocs` is pinned in the `Makefile` (`TFPLUGINDOCS_VERSION`) and run via
`go run <module>@<version>`, so it is not a dependency of this module and
`go.mod` does not know about it. Pinning matters: its rendering changes between
releases, and an unpinned version would make `docs-check` fail on someone
else's machine for no reason. The current pin needs Go 1.25.8 or later, which
`.tool-versions` satisfies.

Three inputs feed a generated page:

| Input | Becomes |
|---|---|
| the schema `Description` on every attribute | the argument reference table |
| `templates/resources/<name>.md.tmpl` | hand-written prose around it (optional) |
| `examples/resources/<name>/resource.tf` | the "Example Usage" section |
| `examples/resources/<name>/import.sh` | the "Import" section |

Data sources use `templates/data-sources/` and
`examples/data-sources/<name>/data-source.tf`.

Without a template, `tfplugindocs` falls back to a default layout that still
picks up the example and the import script. **A weak generated page almost
always means a weak schema `Description`, not a generation problem** — fix it in
the schema, where it also reaches `terraform providers schema -json` and editor
tooling, rather than by hand-writing the page.

Every attribute must carry a `Description`; `TestSchemaDescriptionsPresent`
enforces it. Enum attributes must additionally list their permitted values in
the description — build the list from the `_LOOKUP_ARR` so it cannot drift, and
`TestEnumDescriptionsListValues` will check it.

## Adding a resource

The definition of done is **S1–S8** in
[PLAN.md § "The unit of work"](./PLAN.md#the-unit-of-work): client struct and
CRUD, version gates, schema with a description on every field, CRUD funcs plus
`ImportStatePassthrough` and registration in `provider/provider.go`, a data
source where a lookup-by-name is useful, an acceptance test with an import step,
a sweeper, and docs plus an example.

Collections have their own bar, **C1–C7**, in the same section: a collection
attribute is not tested until it has been tested *plural*. Every collection bug
found in this project so far hid behind a fixture with exactly one element.

### The item / prototype / LLD triad

The bulk of the provider is item definitions. Each item *backend type* (agent,
snmp, http, trapper, simple, external, internal, snmptrap, calculated,
dependent) is exposed as up to three Terraform resources built from one
`resource_<type>_common.go`:

- `zabbix_item_<type>` — a normal item
- `zabbix_proto_item_<type>` — an item prototype, belonging to an LLD rule
- `zabbix_lld_<type>` — an LLD discovery rule

That file supplies a **mod func** (Terraform state → Zabbix struct) and a **read
func** (Zabbix struct → Terraform state); `common_item.go` and `common_lld.go`
wrap them into CRUD closures and merge the shared schema fragments.
`resource_snmp_common.go` is the fullest example. Register the three
constructors in `provider/provider.go` and you are done.

### Version gates

`zabbix.NewAPI` probes `apiinfo.version` before authenticating and stores the
result as an integer — `major*10000 + minor*100 + patch`, so 7.4.13 is 70413.
Gate on the named constants in `internal/zabbix/base.go`, never on bare
integers or a version string:

| Constant | | Covers |
|---|---|---|
| `zabbix.V62` | 6.2 | template groups split from host groups |
| `zabbix.V64` | 6.4 | bearer auth; template vendor fields |
| `zabbix.V70` | 7.0 | proxy model rewrite, `monitored_by`, LLD header arrays |
| `zabbix.V72` | 7.2 | `selectHostGroups`/`selectTemplateGroups` replace `selectGroups` |
| `zabbix.V74` | 7.4 | LLD rule prototypes, template `readme`/`wizard_ready` |

6.0 is the floor, so **no gate below `V62` is meaningful** — do not add one.

Two things that are easy to get wrong here, both learned empirically:

- **Strict parameter validation arrived in 7.0, not 7.2, and is per-method.**
  `item.create`, `itemprototype.create` and `discoveryrule.create` reject
  unknown object properties from 7.0; `host.create` and `graph.create` are still
  lenient even on 8.0.
- **`.get` methods ignore unknown parameters on every version.** A stale
  `selectGroups` against 7.2+ is therefore a *silent wrong answer*, not an
  error — far more dangerous than a hard failure.

### Collections: `TypeList` is a claim that order is semantic

Use `TypeList` only when the Zabbix API guarantees an order *and* that order
carries meaning. Otherwise use `TypeSet`: the server's return order is an
implementation detail that changes between versions, which is exactly how the
8.0 graph regression happened. Do not "fix" an ordering mismatch by sorting the
read result — a `TypeList` must match the *config's* order, which need not agree
with any field the server sorts by.

| Collection | Type |
|---|---|
| item / LLD `preprocessing` | `TypeList` — steps execute in sequence |
| host `inventory` | `TypeList` — a single nested block, not a collection |
| graph `item`, host `interface`, LLD `condition` | `TypeSet` |
| trigger `dependencies`, `tag`, `macro` | `TypeSet` |

**A `TypeSet` hash must cover every user-settable attribute of its element.**
`helper/schema`'s `diffSet` short-circuits on the hash-code lists alone and
never compares elements when they match, so an attribute left out of the hash
can never be seen to change and the edit is *silently discarded* — no diff, no
API call, no error. `hashElementExcept` in `provider/utils.go` does this
generically; exclude only server-assigned ids, which a configuration does not
have and which would otherwise replace every element on every plan.

### The enum lookup-table idiom

Enum-like fields map between human-friendly Terraform strings and Zabbix's
numeric codes through a forward map plus a *generated* reverse map and value
list:

```go
var SNMP_LOOKUP     = map[string]zabbix.ItemType{...}
var SNMP_LOOKUP_REV = map[zabbix.ItemType]string{} // filled by an init-style anon func
var SNMP_LOOKUP_ARR = []string{}                   // validation.StringInSlice + the description
```

The `_REV` and `_ARR` are populated by a package-level
`var _ = func() bool { ... }()` block, so validation messages, reverse lookups
and documentation cannot drift from the forward map.
`TestEnumLookupTablesInSync` checks all three halves agree, and
`TestEnumDescriptionsListValues` checks the description lists what the validator
accepts. Follow the idiom for any new enum: an inline `[]string` in a
`ValidateFunc` is how `zabbix_host.interface.type` ended up needing a bespoke
test of its own.

## Logging

Logging goes through the package-level `log` in `provider/log.go`, emitting
`[TRACE]` / `[DEBUG]` / … lines. Enable with `TF_LOG=TRACE`.

## Releasing

**[RELEASING.md](./RELEASING.md) is the runbook** — the one-time sequence that
turns the v2 line live (Actions, the three workflows, the tag scoping, the
signing secrets, the Registry) and the short checklist for every release after
it. [MAINTAINING.md](./MAINTAINING.md) covers the calendar-driven half: the
nightly, dependency updates and adding a new Zabbix version.

Releases are cut by [goreleaser](https://goreleaser.com) from a `v2.*` tag.
`.goreleaser.yml` builds `{{ .ProjectName }}_v{{ .Version }}` — the binary name
the Terraform Registry requires — for freebsd/windows/linux/darwin on amd64 and
arm64, and signs the checksum file with `GPG_FINGERPRINT`.

**GitHub Actions is currently disabled repository-wide**, and
`.github/workflows/release.yml` is separately neutered on the `v2` branch
(manual dispatch, typed confirmation, plus a guard refusing to run from `v2`).
Do not restore the tag trigger without reading
[PLAN.md § "Branch and release strategy"](./PLAN.md#branch-and-release-strategy):
an unscoped `v*` tag trigger would let this branch publish over the `v0.x`
release line.

## Commit and PR conventions

Fuller version, with the reasoning, in
[CONTRIBUTING.md](./CONTRIBUTING.md#ground-rules).

- `master` and `testenv` are **frozen**. All work happens on `v2`.
- `gofmt -l .` must print nothing before every commit.
- `go build ./...`, `go vet ./...` and `go test ./provider/` must pass; the
  acceptance matrix must be green on 6.0, 7.0 and 7.4 before a release.
- One reviewed change per fix. Thirty-five defects are listed in the v2.0.0 changelog, and they were found during the v2 revival
  and each landed on its own.
- User-visible changes get a [CHANGELOG.md](./CHANGELOG.md) entry under
  `## [Unreleased]`, in the same commit.
