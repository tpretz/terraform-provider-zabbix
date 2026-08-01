# Changelog

All notable changes to this project will be documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)

## [Unreleased]

Verified against Zabbix 7.0.26 (API 7.0.26): all acceptance tests pass and a
full OpenTofu `init` → `plan` → `apply` → `plan` → `apply` → `destroy` cycle
completes with an empty plan after each apply.

### Added

#### API token authentication

- **`api_token` provider argument** (env `ZABBIX_API_TOKEN` / `ZABBIX_TOKEN`).
  Credentials are sent in the HTTP `Authorization: Bearer` header, which Zabbix
  supports from 6.0 and which is the only transport left in 7.2 where the `auth`
  request field was removed. Session ids obtained from `user.login` are sent the
  same way on Zabbix >= 6.0.
  This is required for accounts protected by MFA, because `user.login` cannot
  complete an MFA challenge.
- `username` and `password` are now **optional**; supply either an `api_token` or
  both of them. `password` is now marked sensitive. Token authentication skips
  the login round trip entirely and validates the credential eagerly, so a bad
  token fails at provider configuration rather than on the first resource.
- **`zabbix_token` resource** for managing API tokens. The secret is only
  returned by `token.generate`, once, so it is captured at create time and
  exported as the sensitive `token` attribute.
- **`data.zabbix_token`** for referencing an existing token's metadata (owner,
  expiry, last access) without its secret. The data source has no `token`
  attribute at all, so a secret cannot leak into state through it. Token names
  are unique per user rather than globally, so `userid` narrows the lookup.

#### New resources

- **Access management**: `zabbix_user`, `zabbix_usergroup`, `zabbix_role`
  (plus `data.zabbix_usergroup` and `data.zabbix_role` for referencing the
  builtin groups and roles).
- **Alerting**: `zabbix_mediatype` (email, script and webhook types),
  `zabbix_script`, `zabbix_action` (trigger, discovery, autoregistration,
  internal and service event sources, with operations, recovery operations and
  update operations).
- **Operational**: `zabbix_proxy` as a managed resource (previously only a data
  source), `zabbix_proxygroup` (new in Zabbix 7.0), `zabbix_maintenance` with
  one-time, daily, weekly and monthly periods.
- **Monitoring configuration**: `zabbix_valuemap` (+ `data.zabbix_valuemap`),
  `zabbix_global_macro`, `zabbix_regexp`.
- **Business services and web monitoring**: `zabbix_service`, `zabbix_sla`,
  `zabbix_httptest`.
- **`zabbix_templategroup` resource and data source**: Zabbix 6.2 split template
  groups out of host groups into a separate entity with its own ID space.
  Without this, `zabbix_template` could not be created on Zabbix >= 6.2 at all
  (`Invalid parameter "/1/groups/1": object does not exist`). On Zabbix < 6.2 the
  resource transparently falls back to the host group API so the same
  configuration works across versions.
- **`data.zabbix_server`**: New data source exposing the connected Zabbix server
  version string (e.g. `"7.0.26"`).
- **`zabbix_template_link`**: New resource for declaratively managing the items,
  triggers, and LLD rules belonging to a Zabbix template. Supports Terraform
  import by template ID.
- **Host tags**: `zabbix_host` now supports `tag` blocks.
- **`Makefile`**: `make build`, `make vet`, `make install`, `make testacc`
  (against an already running Zabbix), `make test60` and `make test70`.
- **`docker-compose.zabbix70.yml`**: Compose file for running Zabbix 7.0 LTS
  acceptance tests locally.
- **`CONTRIBUTING_RESOURCES.md`**: the conventions new resources must follow,
  derived from the bugs listed below.
- **Acceptance test coverage** for previously untested resource families:
  template groups, templates, item types, item preprocessing and tags, triggers,
  graphs, LLD rules with prototypes, data source lookups, and every resource
  added above.
- **`examples/e2e/` and `examples/e2e-extended/`**: reproducible end-to-end
  configurations used to verify idempotency with OpenTofu.

### Fixed

#### Zabbix API compatibility

- **Login parameter**: `user.login` now uses `username` for Zabbix >= 5.4.0 and
  `user` for older versions.
- **`apiinfo.version` must be unauthenticated**: Zabbix rejects it outright if
  either the `auth` field or the `Authorization` header is present.
- **`proxy_hostid` → `proxyid`**: `zabbix_host` sends `proxyid` on Zabbix >= 7.0.
  Both spellings are handled on read for backward compatibility.
- **`data_type` on items**: removed from the Item API in Zabbix 5.4; no longer
  serialized.
- **`delta` on items**: removed from the Item API in Zabbix 5.0; no longer
  serialized. Previously caused `unexpected parameter "delta"` on every item
  create.
- **`hosts` on items**: this read-only field (populated by `selectHosts`) was
  serialized on write, causing `unexpected parameter "hosts"`.
- **`discoveryRule` on items**: the struct tag used `omitEmpty` instead of the
  lowercase `omitempty` that `encoding/json` recognises, so this read-only field
  was always serialized.
- **Immutable fields sent on update**: `item.update`, `itemprototype.update`,
  `discoveryrule.update` and `token.update` reject `hostid`, `ruleid` and
  `userid` respectively; they are no longer sent. `action.eventsource` and
  `script.scope` were being cleared to an empty string and serialized anyway,
  producing `an integer is expected`.
- **Preprocessing `error_handler`**: defaulted to an empty string, which
  `omitempty` then stripped, producing `the parameter "error_handler" is
  missing`. Now defaults to `"0"`.
- **LLD `delay` for non-polling types**: `delay` was in the shared LLD schema
  with a default of `3600`, but Zabbix requires `delay` to be `0` for trapper
  and dependent rules, making `zabbix_lld_trapper` and `zabbix_lld_dependent`
  unusable. `delay` is now only exposed on LLD types that poll, mirroring how
  item resources already work.
- **LLD filter `formulaid`**: read back from the API and re-sent on update, which
  Zabbix rejects unless `evaltype` is a custom expression. It is now only sent
  for `evaltype = "custom"`.
- **Action escalation fields**: `esc_period`, `esc_step_from`, `esc_step_to` and
  `evaltype` only exist for trigger (and, for `esc_period`, service) actions and
  only on regular operations. Sending them on other event sources, or on recovery
  and update operations, is rejected as an unexpected parameter.
- **Action `opmessage.mediatypeid`**: defaulted to `"0"` and was therefore always
  sent, which the "notify all involved" operation types reject.
- **Maintenance time periods**: each recurrence accepts a different subset of
  fields and Zabbix rejects the inapplicable ones even when they hold their
  default value.

#### Terraform correctness

- **`zabbix_graph` update was completely broken**: `graph.update` was called
  without `graphid`, so every update failed with `No permissions to referred
  object or it does not exist!`.
- **`zabbix_template` was not idempotent**: `name` defaults to `host` server
  side but was not marked `Computed`, so a non-empty plan remained after every
  apply.
- **`zabbix_graph` was not idempotent**: `ymin_itemid` and `ymax_itemid` are
  reported as `"0"` by Zabbix but were not marked `Computed`, producing a
  permanent diff.
- **`zabbix_script.parameters` was not idempotent**: modelled as a list, but
  Zabbix does not preserve the order, so the parameters appeared reordered on
  every read. It is now a set.
- **Numeric enums replaced with readable values**: `zabbix_service.algorithm`
  and `propagation_rule`, `zabbix_sla.period` and `status`, and
  `zabbix_httptest.status` and `authentication` accepted raw unvalidated numbers.
  They now take readable names and are validated, matching the rest of the
  provider. `zabbix_httptest.verify_peer` and `verify_host` are now booleans.

### Known limitations

- **`zabbix_token` refuses `terraform import`.** Zabbix discloses a token secret
  only at generation time, so an imported token would be held in state without
  its secret, and because that attribute is computed Terraform would report no
  drift. The importer therefore fails with an explanation pointing at
  `data.zabbix_token` (to reference an existing token) or at creating a new
  resource (if a usable secret is needed). Every other resource supports import
  by Zabbix ID.
- Importing a `zabbix_user` leaves `passwd` absent, since Zabbix never returns
  passwords; the first apply after import sets the password to whatever the
  configuration specifies.
- Destroying a configuration that manages its own API token (or the user that
  owns it) while authenticating with that same token fails part way through:
  the credential is deleted while still in use. Use separate credentials to
  destroy such a configuration.
- An API token's role must have `api_access = true`, and managing global objects
  such as global macros and regular expressions requires a `super_admin` role.
- Only Zabbix 7.0 is covered by the test suite. Compatibility code for 5.x and
  6.x is retained but unverified in this fork.

## [0.17.0] — upstream tpretz baseline

Baseline from `github.com/tpretz/terraform-provider-zabbix` at the time of
forking. See upstream repository for prior history.
