# Security policy

## Reporting a vulnerability

Report privately, not as a public issue: use GitHub's
[private vulnerability reporting](https://github.com/tpretz/terraform-provider-zabbix/security/advisories/new)
on this repository.

This is a volunteer-maintained provider. Expect an acknowledgement within a week and a
fix in the next release rather than an out-of-band one, unless the issue is being
actively exploited.

## Supported versions

Only the latest `v2.x` release. The `v0.x` line is unmaintained and cannot talk to
Zabbix 7.2 or later at all — if you are on `v0.x`, upgrading is the fix. See
[MIGRATING.md](./MIGRATING.md).

## What this provider does with your credentials

Worth knowing before you file, because two of these look like bugs and are not:

- **Terraform state holds secrets in cleartext.** That is Terraform's design, not this
  provider's choice. Any attribute the provider reads back from Zabbix is in your state
  file — including SNMPv3 passphrases and IPMI passwords. Use a state backend with
  encryption and access control. `Sensitive: true` redacts *plan output*, not state.
- **Pre-shared keys are write-only.** `tls_psk` and `tls_psk_identity` on hosts and
  proxies are sent but never read back, because `proxy.get` and `host.get` do not return
  them. They are therefore not in state, and `terraform import` cannot recover them.
- **Request and response bodies are redacted before logging.** Passwords, session
  tokens, PSKs, IPMI passwords and SNMPv3 passphrases are replaced with
  `***REDACTED***` in `TF_LOG` output. If you find a credential in a log, that is a
  vulnerability — please report it.
- **SNMPv3 passphrases are not marked sensitive, on purpose.** Terraform
  propagates sensitivity to everything derived from the containing block, so
  marking them taints every expression over `interface` — including the
  interface `id`, which carries no secret and which every agent or SNMP item
  needs. The schema defaults them to user-macro references, which is the right
  way to hold them; a literal in the configuration will appear in plan output.
- **`tls_insecure` disables certificate *and* hostname verification**, which means the
  provider will send its credentials to whatever answers on that address. It exists for
  testing against a self-signed lab server. There is deliberately no environment-variable
  fallback for it.

## Scope

In scope: credential disclosure, anything that lets a configuration reach a Zabbix
object it should not, dependency vulnerabilities reachable from provider code.

Out of scope: vulnerabilities in Zabbix itself (report those to
[Zabbix](https://www.zabbix.com/security)), and Terraform's plaintext state, which is
upstream behaviour described above.
