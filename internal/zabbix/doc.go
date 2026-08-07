/*
Package zabbix is the Zabbix JSON-RPC API client used by terraform-provider-zabbix.

It began life as the standalone library github.com/tpretz/go-zabbix-api, vendored
here as a git submodule and merged into the provider in v2. It is deliberately an
internal package: the provider is its only consumer, and it is free to change shape
whenever the provider needs it to.

# Supported versions

Zabbix 6.0 LTS and above. Everything below that was removed in v2 — see MIGRATING.md.
Behaviour that differs between versions is gated on Config.Version, which NewAPI fills
in from APIInfo.version as major*10000 + minor*100 + patch (7.4.13 is 70413). Use the
named constants V62, V64, V70, V72 and V74 rather than bare integers.

# Authentication

NewAPI probes APIInfo.version unauthenticated. After that, Login stores a session token
in API.Auth, or the caller may set API.Auth directly to a pre-created API token. From
6.4 the token travels in an Authorization: Bearer header; below that it is sent as the
JSON-RPC "auth" body property, which 7.2 removed.

# Testing

This package has no tests of its own. It is covered end to end by the provider's
acceptance suite, which exercises every call path against live Zabbix 6.0, 7.0, 7.4 and
8.0 servers — see TESTING.md. The library's original TEST_ZABBIX_* test harness was
deleted in v2: it had not compiled in years (a nested go.mod hid it from ./...), used an
environment convention the provider suite does not share, and duplicated coverage the
acceptance tests already provide against four real servers instead of one.
*/
package zabbix
