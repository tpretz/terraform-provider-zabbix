# Releasing

How to cut a release of this provider, and — for `v2.0.0` specifically — how to
turn on the automation that has been deliberately switched off for the whole of
the v2 development line.

Read this in full before starting the v2.0.0 release. It is a one-way door in
two places: publishing to the Terraform Registry cannot be undone (a version
can be deprecated, never withdrawn), and a badly scoped tag trigger can publish
this branch over the `v0.x` release line.

> **The single most dangerous mistake available here** is restoring
> `release.yml`'s original `on: push: tags: ['v*']` trigger. That pattern
> matches `v0.17.1` as happily as `v2.0.0`, so any stray `v0.*` tag — or a
> re-tag of the frozen line — would build from whatever commit it names and
> publish over releases people are already pinned to. Scope it to `v2.*`, and
> keep a job-level check as well. [Step 5](#5-releaseyml-restore-the-tag-trigger-scoped-to-v2)
> does both.

---

## Part 1 — cutting `v2.0.0`

Everything in Part 1 is done once. Later releases are [Part 2](#part-2--every-release-after-v200).

Current state, which every step below assumes:

| | |
|---|---|
| GitHub Actions | **disabled repository-wide** |
| `ci.yml` | `workflow_dispatch` only, job guard refusing `refs/heads/v2` |
| `nightly.yml` | `workflow_dispatch` only, job guards refusing `refs/heads/v2` |
| `release.yml` | `workflow_dispatch` only, typed `confirm: RELEASE` input, job guard refusing `refs/heads/v2` |
| `dependabot.yml` | present, but read from the default branch — which is still `master` |
| default branch | `master`, last released `v0.17.0` |
| `v2` | not pushed |

### 0. Pre-flight, locally

```bash
git rev-parse --abbrev-ref HEAD        # must print: v2
git status --porcelain                 # must print nothing
gofmt -l .                             # must print nothing
make check-toolchain
make build vet test
go test ./...                          # `make test` covers ./provider only;
                                       # internal/zabbix's redaction tests are
                                       # not in it, and not in ci.yml either
make docs-check                        # docs/ must be current with the schema;
                                       # ci.yml does NOT run this, so it is on you
make testacc                           # 6.0 + 7.0 + 7.4 — the release gate
make testall                           # adds 8.0; read it, do not gate on it
```

`make testacc` must be green. A release does not go out on a red matrix on any
gating version; 8.0 is reported only.

Then the release-content checks:

```
[ ] CHANGELOG.md: the 2.0.0 heading's date is set (it ships as "UNRELEASED")
[ ] CHANGELOG.md: every user-visible change since v0.17.0 is in it
[ ] MIGRATING.md: still accurate end to end — it is the first thing an existing
    user reads, and every breaking change in the changelog links into it
[ ] README.md and templates/index.md.tmpl: the version support table is current
[ ] no `replace` directive in go.mod (there is none; the submodule was retired)
```

### 1. Verify the release build locally

**This is the step that has never been run before this release.** Do not skip
it: the first real execution of `.goreleaser.yml` should not be the one that is
also publishing.

```bash
goreleaser check                       # config schema
goreleaser build --snapshot --clean    # all eight platform binaries, no publish
ls dist/*/
```

Expected: eight targets (freebsd/windows/linux/darwin × amd64/arm64), each
containing a binary named **`terraform-provider-zabbix_v<version>`**. That name
is not cosmetic — the Terraform Registry requires exactly
`terraform-provider-{name}_v{version}`, and a mismatch produces a version the
Registry accepts and Terraform then cannot use. Takes about a minute.

`goreleaser build` deliberately does not sign or publish. `--clean` wipes
`dist/`, which is git-ignored.

Two known, harmless things you will see:

- The snapshot version is derived from the last reachable tag (`0.15.0-SNAPSHOT-<sha>`
  on this branch). Snapshot versions are not real; ignore it.
- `.goreleaser.yml` passes `-X main.version=… -X main.commit=…`, but `main.go`
  declares neither variable. The linker ignores `-X` for symbols that do not
  exist, so the flags are a silent no-op. Not worth fixing during a release;
  either add the variables or drop the flags afterwards.

Optionally, smoke-test the built binary against a real Terraform run using a
`dev_overrides` block in `~/.terraformrc` pointing at `dist/<target>/`, with one
of the configurations in `examples/` and a stack from `make testenv-up-74`.

### 2. Push the branch

```bash
git push -u origin v2
```

Nothing runs: Actions is still off repository-wide, and every workflow is
`workflow_dispatch`-only with a `refs/heads/v2` guard on top.

### 3. Enable GitHub Actions repository-wide

Settings → Actions → General → *Allow all actions and reusable workflows*.

```bash
gh api repos/tpretz/terraform-provider-zabbix/actions/permissions
# {"enabled":true,...}
```

Still nothing runs — the per-workflow guards are what has been holding the line,
and they are all still in place. Removing them is steps 4–6.

### 4. `ci.yml`: restore the triggers, drop the guard

Edit `.github/workflows/ci.yml`:

- rename the workflow from `ci (disabled on v2)` to `ci`
- delete the `DISABLED ON THE v2 BRANCH` header block
- replace the `workflow_dispatch:`-only trigger with:

  ```yaml
  on:
    push:
      branches: [v2]
    pull_request:
    workflow_dispatch:
  ```

  `branches: [v2]` and not `[main]` — the comment block in the file predates the
  branch decision. Update it to whatever the default branch is at the time.
- delete the job-level `if: github.ref != 'refs/heads/v2'`

Commit and push, then confirm it actually ran and was green:

```bash
gh run list --workflow=ci.yml --limit 3
```

Do this **before** the release workflow. CI going green on a real push is the
cheapest possible proof that enabling Actions did not surface something the
local build hides.

### 5. `release.yml`: restore the tag trigger, scoped to `v2.*`

Edit `.github/workflows/release.yml`:

- rename the workflow from `release (disabled on v2)` to `release`
- delete the `DISABLED ON THE v2 BRANCH` header block
- replace the `workflow_dispatch` block, including its `confirm` input, with:

  ```yaml
  on:
    push:
      tags:
        - 'v2.*'
  ```

- replace the job-level guard — do **not** simply delete it. It becomes the
  second half of the tag scoping:

  ```yaml
      if: startsWith(github.ref, 'refs/tags/v2.')
  ```

Two independent checks for the same thing is deliberate. The trigger pattern is
easy to widen by accident while editing (`v*` is what was there originally and
what most provider repositories use); the job guard means that a widened
pattern still cannot publish a `v0.*` or `v1.*` tag.

Verify the result reads as intended before pushing:

```bash
python3 -c "import yaml,glob;[print(f, list((lambda d: d[True] if True in d else d['on'])(yaml.safe_load(open(f)))), [j.get('if') for j in yaml.safe_load(open(f))['jobs'].values()]) for f in sorted(glob.glob('.github/workflows/*.yml'))]"
```

`release.yml` must show its trigger as `['push']` with the tag filter `v2.*`,
and the job `if` must still be present. If either the filter or the `if` is
missing, stop.

### 6. `nightly.yml`: restore the schedule, drop the guards

- rename the workflow from `nightly acceptance (disabled on v2)` to
  `nightly acceptance`
- restore the `schedule:` trigger from the header comment (`cron: '17 3 * * *'`),
  keeping `workflow_dispatch:`
- delete the `if: github.ref != 'refs/heads/v2'` from the `acceptance` job, and
  the same clause from the `report` job's condition — keeping
  `always() && needs.acceptance.result == 'failure'`

Then dispatch it once by hand and let it finish, before tagging:

```bash
gh workflow run nightly.yml
gh run watch
```

A green nightly on real runners is worth more than a green local matrix: it is
the first time the acceptance suite has run anywhere but a developer's machine,
and Docker resource limits, port availability and image pull rates are all
different there.

### 7. Confirm the signing secrets

goreleaser signs the checksum file, and the Terraform Registry will reject an
unsigned or wrongly signed release. Two repository secrets are required:

```bash
gh secret list
# GPG_PRIVATE_KEY   updated ...
# PASSPHRASE        updated ...
```

- `GPG_PRIVATE_KEY` — the ASCII-armoured **private** key, exported with
  `gpg --armor --export-secret-keys <key-id>`
- `PASSPHRASE` — that key's passphrase

**The GPG import action was replaced during this release.** The old
`hashicorp/ghaction-import-gpg` is archived, and its last release is a Node 12
action that GitHub no longer executes at all — it would have failed outright,
not deprecated gracefully. It is now `crazy-max/ghaction-import-gpg@v6`, which
takes the key and passphrase as `with:` inputs rather than `env:` and still
exports the `fingerprint` output the goreleaser step consumes. That path has
therefore **never run**, which is a second reason step 8 watches the run rather
than tagging and walking away.

Also confirm the *public* half of the same key is still registered against the
`tpretz` namespace in the Terraform Registry (Registry → User settings →
Signing keys). It was, for the `v0.x` releases; if the key has been rotated
since, upload the new one **before** tagging, or the Registry will ingest the
release and then reject its signature.

### 8. Tag and publish

```bash
git tag -a v2.0.0 -m "v2.0.0"
git push origin v2.0.0
gh run watch                            # follow the release workflow
```

When it finishes:

```bash
gh release view v2.0.0 --json assets --jq '.assets[].name'
```

Expected assets: eight `terraform-provider-zabbix_2.0.0_<os>_<arch>.zip`, plus
`terraform-provider-zabbix_2.0.0_SHA256SUMS`, `…_SHA256SUMS.sig` and
`terraform-provider-zabbix_2.0.0_manifest.json`. A missing `.sig` means the GPG
step did not do its job — fix it and re-cut as `v2.0.1` rather than moving the
tag. A missing `manifest.json` means `.goreleaser.yml`'s `release.extra_files`
did not fire; the Registry then assumes protocol 5.0, which is right today and
silently stops being right if this provider ever moves to
`terraform-plugin-framework`.

`.goreleaser.yml` sets `changelog: disable: true`, so the GitHub release body is
empty by design. Paste the `2.0.0` section of `CHANGELOG.md` into it, or at
minimum link to `CHANGELOG.md` and `MIGRATING.md` — for a release this breaking,
an empty body is a support burden.

### 9. Verify the Terraform Registry picked it up

Ingestion is webhook-driven and usually takes a few minutes.

```bash
curl -s https://registry.terraform.io/v1/providers/tpretz/zabbix/versions \
  | python3 -m json.tool | grep -A2 '"2.0.0"'
```

Then the real test, in a scratch directory:

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

```bash
terraform init      # must download and verify 2.0.0
```

If the Registry does not pick it up, re-sync from Registry → Settings → the
provider → *Resync*. The usual causes are an unsigned checksum file, a signing
key that is not registered, or a binary whose name does not match
`terraform-provider-zabbix_v2.0.0`.

The documentation pages come from `docs/` in the tagged commit. Check that
`https://registry.terraform.io/providers/tpretz/zabbix/latest/docs` renders the
index and that the sidebar shows the seven subcategories; if a page is missing,
`docs/` was stale at tag time, which `make docs-check` in step 0 exists to
prevent.

### 10. Make `v2` the default branch, and leave `master` as an archive

```bash
gh repo edit tpretz/terraform-provider-zabbix --default-branch v2
```

This is what activates **Dependabot** — it reads `.github/dependabot.yml` from
the default branch, so until now it has been reading `master`, which has none.
Expect its first pull requests within a day; [MAINTAINING.md](./MAINTAINING.md#reviewing-a-dependabot-pull-request)
covers reviewing them.

For `master`:

- do **not** run `gh repo archive` — that archives the *repository*, making the
  whole thing read-only. What is wanted is a dormant branch, not a dormant
  project.
- add a branch protection rule on `master` that allows nobody to push, or simply
  leave it alone. It is frozen by policy; the tags on it are what people are
  pinned to and must keep resolving.
- do not delete it, and do not delete or move any `v0.*` tag. Existing
  `terraform init` runs resolve against them.

Optionally open a tracking issue on `master` pointing at v2, for anyone who
lands there from an old link.

### 11. Post-release

```
[ ] CHANGELOG.md: open a fresh `## [Unreleased]` section
[ ] PLAN.md: tick Phase 6 and record the release
[ ] the repository description and topics still describe the provider accurately
[ ] the nightly ran successfully on its own schedule at least once
```

---

## Part 2 — every release after v2.0.0

Once Part 1 is done the process is short, because the automation is on.

```
[ ] make testacc is green (and read make testall)
[ ] gofmt -l . prints nothing; make build vet test docs-check pass
[ ] CHANGELOG.md: move `Unreleased` into a dated version heading
[ ] MIGRATING.md gains a section if anything breaks
[ ] goreleaser build --snapshot --clean still succeeds
[ ] git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z
[ ] gh run watch, then check the release assets include SHA256SUMS.sig
[ ] paste the changelog section into the GitHub release body
[ ] confirm the Registry lists the new version
[ ] open a fresh `## [Unreleased]`
```

Version numbering follows semver against the *provider's* surface:

| Change | Bump |
|---|---|
| a new resource, data source or optional attribute | minor |
| a bug fix that does not change a schema | patch |
| an attribute removed or renamed, a type changed (`TypeList`→`TypeSet`), a required attribute added | **major**, with a `MIGRATING.md` section |
| **raising the Zabbix version floor** | **major** — gated code is deleted, not merely untested. See [MAINTAINING.md § step 8](./MAINTAINING.md#8-drop-the-version-that-fell-out-of-support) |

A `v2.*` tag is the only thing that publishes. Prereleases are `v2.1.0-rc1`,
which the tag filter matches and the Registry marks as a prerelease.

Never move or delete a published tag. If a release is wrong, cut the next patch.
