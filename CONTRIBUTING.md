# Contributing

Contributions are welcome. This file is about the things a newcomer to *this*
codebase gets wrong — the patterns that look optional and are not, and the four
traps that between them account for most of the bugs found in the v2 revival.

For build, test and documentation mechanics, read
[DEVELOPMENT.md](./DEVELOPMENT.md) first. For the acceptance harness, read
[TESTING.md](./TESTING.md). This file assumes both.

---

## Ground rules

- **All work happens on `v2`.** `master` and `testenv` are frozen: no commits,
  no backports, no re-tagging. The published `v0.x` releases stay as they are.
- **One reviewed change per commit.** Twenty-eight defects are listed in the v2.0.0 changelog, and they were found during the v2
  revival and each landed on its own. A commit that fixes two things is two
  commits.
- **Say what you measured.** This project's commit messages record what was run
  against which Zabbix version and what came back, because almost every
  assumption made about the Zabbix API during the revival turned out to be
  wrong in some version. "Verified on 7.4" is worth more than a paragraph of
  reasoning.
- **A bug found while doing something else is a separate change.** Report it,
  land the change you were making, then fix it on its own.

Before you push:

```bash
gofmt -l .            # must print nothing
make build vet test   # go build ./... , go vet ./... , unit tests
make docs-check       # if you touched a schema, a template or an example
make testacc          # if you touched provider or client code (needs Docker)
```

`make testacc` runs 6.0, 7.0 and 7.4 — the release gate. `make testall` adds
8.0, which is reported but never blocking.

---

## The item / prototype / LLD triad

The bulk of the provider is item definitions, and this is the pattern to
understand before touching any of them.

Each item *backend type* — agent, snmp, http, trapper, simple, external,
internal, snmptrap, calculated, dependent — is exposed as up to **three**
Terraform resources, all built from one `provider/resource_<type>_common.go`:

| Resource | What it is |
|---|---|
| `zabbix_item_<type>` | a normal item |
| `zabbix_proto_item_<type>` | an item **prototype**, belonging to an LLD rule |
| `zabbix_lld_<type>` | an LLD **discovery rule** |

That one file provides two callbacks and nothing else:

```go
// Terraform state -> Zabbix struct
func itemTrapperModFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item)

// Zabbix struct -> Terraform state
func itemTrapperReadFunc(d *schema.ResourceData, m interface{}, item *zabbix.Item)
```

LLD variants take `*zabbix.LLDRule` and are named `lld<Type>ModFunc` /
`lld<Type>ReadFunc`.

`common_item.go` and `common_lld.go` turn those into CRUD closures —
`itemGetCreateWrapper(mod, read)`, `protoItemGetReadWrapper(read)`,
`lldGetUpdateWrapper(mod, read)` and so on — and `buildItemObject` fills in
every field the types share. Schemas are assembled with
`mergeSchemas(itemCommonSchema, itemDelaySchema, …, typeSpecificSchema)`.

So a new backend type is one file plus three registrations in
`provider/provider.go`. `resource_snmp_common.go` is the fullest example.

Three things that catch people:

- **The mod and read funcs must be exact inverses.** Anything the mod func
  sends and the read func does not restore is a permanent diff; anything the
  read func sets that the mod func cannot express is a diff the user cannot
  apply away. The `post_type` bug — a default of `"body"`, which is not one of
  that attribute's values — was exactly this, on every HTTP item.
- **The read path is shared and must stay total.** `d.Set` panics on a key the
  schema does not declare, so a shared read function cannot set an attribute
  that only some resources have. That panic shipped once: the `zabbix_template`
  data source crashed on every read because `templateRead` set `templates`,
  which the data source schema did not declare. Where a type genuinely differs,
  give it a *pinned* schema rather than removing the attribute — `zabbix_lld_trapper`
  and `zabbix_lld_dependent` keep a `delay` attribute validated to `"0"`
  precisely so the shared read path stays total.
- **One `zabbix.Item` struct serves all ten backend types**, and from Zabbix
  7.0 `item.create` rejects properties that do not apply to the type. Adding a
  field to `Item` without `omitempty` sends it on every item of every type. See
  [the `omitempty` trap](#the-omitempty-trap), which is the other half of this.

---

## Version gating

`zabbix.NewAPI` probes `apiinfo.version` before authenticating and stores the
result as an integer.

**The encoding is `major*10000 + minor*100 + patch`.** So:

| Version | Encoded |
|---|---|
| 6.0.48 | `60048` |
| 6.2 | `60200` |
| 7.4.13 | `70413` |
| 8.0 | `80000` |

This project's own documentation got that wrong once — it said 6.2 was `62000`
— and the mistake is invisible until it costs you a debugging round: `62000` is
version 6.20, a release that will never exist, so the gate simply never fires
and the code behind it is dead. If you are writing a number here, you are doing
it wrong anyway:

```go
if api.Config.Version >= zabbix.V72 { … }   // yes
if api.Config.Version >= 70200 { … }        // no
if api.Config.Version >= 72000 { … }        // no, and it can never be true
```

The constants live in `internal/zabbix/base.go`:

| Constant | | Covers |
|---|---|---|
| `zabbix.V62` | 6.2 | template groups split from host groups |
| `zabbix.V64` | 6.4 | bearer auth; template `vendor_name`/`vendor_version` |
| `zabbix.V70` | 7.0 | proxy model rewrite, `monitored_by`, LLD header arrays |
| `zabbix.V72` | 7.2 | `selectHostGroups`/`selectTemplateGroups` replace `selectGroups` |
| `zabbix.V74` | 7.4 | LLD rule prototypes, template `readme`/`wizard_ready` |

6.0 is the floor, so **no gate below `V62` is meaningful** — do not add one.
Add a new constant rather than a literal when a new version needs one; see
[MAINTAINING.md](./MAINTAINING.md#4-gate-the-deltas).

Two findings that are easy to re-derive the hard way, both measured against
live servers:

- **Strict parameter validation arrived in 7.0, not 7.2, and is per-method.**
  `item.create`, `itemprototype.create` and `discoveryrule.create` reject
  unknown object properties from 7.0; `host.create` and `graph.create` are
  still lenient even on 8.0.
- **`.get` methods ignore unknown parameters on every version.** A stale
  `selectGroups` against 7.2+ is therefore a *silent wrong answer* — the
  resource reads back with no groups and the tests may stay green. This is more
  dangerous than a hard failure, and it is why reading the upstream changes
  page is a step in the maintenance runbook rather than an optional extra.

---

## The `_LOOKUP` / `_REV` / `_ARR` idiom

Enum-like fields map between human-friendly Terraform strings and Zabbix's
numeric codes through a forward map plus a *generated* reverse map and value
list. Never write the reverse map by hand, and never write an inline
`[]string{…}` in a `ValidateFunc`:

```go
var SNMP_LOOKUP     = map[string]zabbix.ItemType{ "snmp": zabbix.SNMPAgent, … }
var SNMP_LOOKUP_REV = map[zabbix.ItemType]string{} // generated
var SNMP_LOOKUP_ARR = []string{}                   // generated

var _ = func() bool {
    for k, v := range SNMP_LOOKUP {
        SNMP_LOOKUP_REV[v] = k
        SNMP_LOOKUP_ARR = append(SNMP_LOOKUP_ARR, k)
    }
    sort.Strings(SNMP_LOOKUP_ARR)
    return false
}()
```

`_ARR` feeds both `validation.StringInSlice` and the attribute's
`Description`, so validation messages, reverse lookups and published
documentation cannot drift from the forward map.

**Sort the `_ARR`.** Go randomises map iteration order, so an unsorted list
makes a validation message — and a generated docs page — differ between builds
of the same source, and `make docs-check` fails at random.
`TRIGGER_PRIORITY_ARR` is ordered by severity code instead of alphabetically,
deliberately, because the list is a scale.

Four unit tests hold this line, and they run without a server:
`TestEnumLookupTablesInSync`, `TestEnumValueListsAreSorted`,
`TestEnumDescriptionsListValues`, `TestSchemaDescriptionsPresent`.

`zabbix_host.interface.type` is the one enum whose *validator* deliberately
still reads an inline slice, so that `TestSchemaHostInterfaceTypeMatchesLookup`
keeps catching drift against `HOST_IFACE_TYPES` rather than becoming trivially
true. Do not copy it.

---

## Collections

### `TypeList` is a claim that order is semantic

Use `TypeList` only when the Zabbix API guarantees an order **and** that order
carries meaning. Otherwise use `TypeSet`. The server's return order is an
implementation detail that changes between versions: Zabbix 8.0 returns a
graph's items in a different order than 6.0/7.x, and LLD filter conditions come
back in submission order on 6.0 but sorted by formula id on 7.4 and 8.0.

Do not "fix" an ordering mismatch by sorting the read result. A `TypeList` must
match the *config's* order, which need not agree with any field the server
sorts by — sorting graph items by `sortorder` fixed nothing and broke 7.4.

| Collection | Type | Why |
|---|---|---|
| item / LLD `preprocessing` | `TypeList` | steps execute in sequence; order is genuinely semantic |
| host `inventory` | `TypeList` | a single nested block, not a collection |
| graph `item`, host `interface`, LLD `condition` | `TypeSet` | server return order is not stable across versions |
| trigger `dependencies`, `tag`, `macro`, `templates`, `groups` | `TypeSet` | |

### A set's hash must cover every user-settable attribute

This is the sharpest trap in the codebase, and it fails **silently**.

`helper/schema`'s `diffSet` short-circuits on
`reflect.DeepEqual(os.listCode(), ns.listCode())` — it compares the two sets'
**hash codes only** and never looks at the elements when the codes match. An
attribute left out of the hash therefore *cannot be seen to change*: editing it
produces an empty plan, no API call and no error. The edit is discarded.

So hashing "just the identifying field", however natural it reads, is wrong.
Hash every user-settable attribute and exclude only server-assigned ids, which
config does not have and which would otherwise make every element look replaced
on every plan. `hashElementExcept` in `provider/utils.go` does that generically,
so a newly added field cannot be forgotten:

```go
Set: func(v interface{}) int { return hashElementExcept(v, "id") },
```

A `TypeSet` with no `Set:` function at all uses the SDK's reflective default,
which is *usually* fine and is still worth replacing with an explicit one: it
reads as intentional, and it is the only place the rule above is visible.

Two consequences to keep in mind:

- An edited element arrives with no id, so where the server refuses a
  delete-and-recreate the provider has to hand the old id back —
  `hostReuseInterfaceIDs` in `resource_host.go` exists because Zabbix will not
  delete an interface that has items bound to it.
- **Sets cannot be indexed from HCL.** `zabbix_host.x.interface[0].id` does not
  parse; `one(...)` is the replacement. This is the v2 change most likely to
  break an existing configuration.

### <a id="the-omitempty-trap"></a>The `omitempty` trap

**Zabbix reads an absent property as "leave this as it is" and `[]` as "clear
it".** An `omitempty` struct tag on a property the user is allowed to empty
therefore means the clear is never sent: the server keeps what it had, the next
read puts it straight back into state, and the user gets a diff that reapplies
forever and never converges.

This produced **six** separate bugs in the v2 release — item `preprocessing`
and `tag`, LLD `preprocessing`, trigger `dependencies` and `tag`, LLD
`macro_path`, HTTP item `posts`/`proxy`, and trigger `url`. Every one of them
was invisible to the tests because no test ever removed the last element.

The rule when adding a field to a struct in `internal/zabbix/`:

| The property | Tag it |
|---|---|
| applies to every object of this type, and can be emptied | **no** `omitempty`; normalise nil to an empty slice on the write path |
| applies only to some object types (an `Item` field only HTTP items have) | `*string` / pointer, not `omitempty` — a pointer to `""` marshals as `""`, a nil one is omitted |
| create and update disagree about whether it may be `[]` | a pointer to a slice type, set explicitly in the `prep*Update` path |

The middle row exists because one `Item` struct serves all ten backend types
and 7.0+ rejects properties that do not apply, so plain `omitempty` removal
would make every agent item send `"posts": ""` and fail outright. The third row
is `LLDRule.MacroPaths`, where 6.0 rejects `[]` on create but accepts it on
update: `nil` omits the key, `&LLDMacroPaths{}` sends `[]`.

And watch the spelling. `discoveryRule` was tagged `omitEmpty`, which
`encoding/json` silently ignores, so it was serialised as `null` on every
single call for years.

---

## Definitions of done

### `S1`–`S8`: a new resource

The full text is in [PLAN.md § "The unit of work"](./PLAN.md#the-unit-of-work).
Summary:

| | Step |
|---|---|
| S1 | client struct and CRUD in `internal/zabbix/<obj>.go` |
| S2 | version gates, named constants, old path kept for every supported version |
| S3 | resource schema, a `Description` on every field, validators, lookup tables |
| S4 | CRUD funcs, `ImportStatePassthrough`, registration in `provider.go` |
| S5 | data source, where a lookup-by-name is useful |
| S6 | acceptance test: create → update → re-read, an import step, a `SkipFunc` for version-bound behaviour, and `C1`–`C7` for every collection |
| S7 | sweeper, so an aborted run self-cleans |
| S8 | docs and a runnable example |

A partial resource is worse than none: it ships an attribute surface users
build on and a maintenance burden nobody signed up for.

### `C1`–`C7`: a collection attribute

**A collection is not tested until it has been tested plural.** Every
collection bug in this project hid behind a fixture with exactly one element —
one element cannot distinguish a set from a list, cannot show an ordering
assumption, and cannot show an identity assumption.

| | Case | What it must do |
|---|---|---|
| C1 | none | omitted entirely (where optional) — creates, plans clean, imports back empty |
| C2 | one | the trivial case |
| C3 | many | three elements where the server may reorder, two otherwise; at least two *of the same kind*, so identity is proven to come from content and not position |
| C4 | reordered | the same elements in a different order, as a `PlanOnly: true` step. A **set** must plan empty; a **list** must produce a diff and the new order must survive the round trip |
| C5 | edit one of many | change one attribute of one element; assert the others are untouched and kept their server-assigned ids |
| C6 | remove one, then all | N → N−1 → 0, and the removal shown **reaching the server** |
| C7 | import at full size | `ImportStateVerify` with the collection at its largest — the only check that the flatten function and the set hash agree |

Two rules that follow:

- **Assert set elements by content** — `TestCheckTypeSetElemNestedAttrs`,
  `TestCheckTypeSetElemAttrPair` — never by index. A set's indices in test state
  are artefacts of the state shim and mean nothing.
- **C6 needs a server-side check.** State is written by the provider's own read,
  so a collection the provider silently drops still looks right there. The
  `testAccCheck*Count` helpers in `provider/acc_collection_test.go` re-read the
  object from Zabbix and count what actually came back.

Shared machinery — `common_tag.go`, `common_macro.go`, the item and LLD
preprocessor blocks — is tested plural **once** per code path, with a
single-element smoke check elsewhere. Where that decision was made is written
next to the collection; the index is in the header comment of
`provider/acc_collection_test.go`. Note that item and LLD preprocessing are
*not* the same code path.

---

## Changelog entries

Add your entry to `## [Unreleased]` in [CHANGELOG.md](./CHANGELOG.md), in the
same commit as the change, under Added / Changed / Deprecated / Removed / Fixed
/ Security. Write it for someone reading release notes, not for someone reading
the diff: what changed for a user, and what they have to do about it. A
breaking change links to its [MIGRATING.md](./MIGRATING.md) section.

Skip it for changes with no user-visible effect — refactors, test-only work,
internal documentation.

**There is deliberately no changelog-fragment tool** (`changie`, `towncrier`,
`.changelog/` files). What those buy is conflict-free merges of a file with many
concurrent writers, and this repository has one maintainer; what they cost is a
per-change YAML file, a second format to learn, a release-time command that has
to be remembered exactly once every year or two, and a `CHANGELOG.md` nobody can
edit by hand any more. Adopt one when there is more than one regular
contributor and `CHANGELOG.md` starts producing merge conflicts — that is the
signal, and until it appears the tool is ceremony.

---

## Documentation

`docs/` is **generated**. An edit made there by hand is thrown away by the next
`make docs`. Three inputs feed a page — the schema `Description`, the prose in
`templates/`, and the HCL in `examples/` — and a weak page is almost always a
weak `Description`. See
[DEVELOPMENT.md § Documentation](./DEVELOPMENT.md#documentation).

Run `make docs-check` before pushing any schema change; it regenerates and fails
if `docs/` moved.

---

## Reporting a bug

Include the **Zabbix version** (`apiinfo.version`, not the frontend's About
page), the provider version, the configuration that reproduces it, and the
relevant part of `TF_LOG=TRACE terraform plan`, which includes the provider's
own `[TRACE]`/`[DEBUG]` lines and the API calls it made.

Version-specific behaviour is the norm here rather than the exception, so a
report without a version number usually cannot be acted on.
