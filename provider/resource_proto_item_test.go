package provider

import (
	"testing"
)

// protoItemFixtureHCL is the scaffolding every zabbix_proto_item_* test needs:
// something to own the prototype (a template) and a discovery rule for it to
// belong to.
//
// Item prototypes are the only member of the item/prototype/LLD triad that
// carries a "ruleid", and nothing else in the suite exercises that wiring --
// the mod/read funcs are shared with the plain item variant, so what these
// tests are really covering is protoItemGetCreateWrapper/Read/Update, the
// itemprototype.* API namespace, and ruleid surviving a round trip through
// selectDiscoveryRule on import.
//
// The rule is a trapper discovery rule deliberately: it is the only LLD type
// that needs neither an interface nor a poll interval, so it adds no
// version-dependent behaviour of its own to a prototype test. Trapper-type
// rules require delay == 0 and Zabbix rejects any other value.
const protoItemFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host = "test-template"
}
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key = "test.lld.rule"
	name = "Test LLD Rule"
	delay = "0"
}
`

// protoItemConfig prepends the shared fixture to a prototype resource block and
// runs the result through hcl() so it works either side of the 6.2 template
// group split.
func protoItemConfig(t *testing.T, body string) string {
	return hcl(t, protoItemFixtureHCL+body)
}
