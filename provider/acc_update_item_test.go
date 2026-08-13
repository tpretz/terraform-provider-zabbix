package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// U1/U2 for the item family (PLAN.md § "The unit of work").
//
// Every attribute an item carries is declared in one of five shared fragments
// in common_item.go, plus one file per backend type, so the coverage here is
// per *fragment* rather than per resource -- see the table in the header of
// acc_update_test.go and the same convention in acc_collection_test.go.
//
// Each test changes every attribute it owns in one step and asserts the result
// against a re-read from Zabbix. Changing them together rather than one at a
// time is deliberate: with expectUpdate on the step, a single ForceNew
// anywhere in the set would turn the whole plan into a replacement and fail
// it, which is the U4 assertion for every attribute at once.

// updateItemTemplateHCL is the two-template fixture the item tests build on.
// The second template exists so the ForceNew tests have somewhere to move to;
// it is unused here and cheap.
const updateItemTemplateHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-update-template-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-update-template"
}
`

// TestAccUpdateItemAgent owns itemCommonSchema (key, name, valuetype,
// history, trends), itemDelaySchema (delay), itemPreprocessorSchema, the
// shared tagSetSchema, and the agent fragment's own "active".
func TestAccUpdateItemAgent(t *testing.T) {
	const addr = "zabbix_item_agent.testitem"

	item := func(body string) string {
		return hcl(t, updateItemTemplateHCL+`
resource "zabbix_item_agent" "testitem" {
	hostid = zabbix_template.testtmpl.id
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: item(`
	key       = "test.update.item.a"
	name      = "Update Item A"
	valuetype = "unsigned"
	history   = "90d"
	trends    = "365d"
	delay     = "1m"
	active    = false
	tag {
		key   = "tag-a"
		value = "value-a"
	}
	preprocessor {
		type   = "multiplier"
		params = [ "10" ]
	}
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"key_":       "test.update.item.a",
					"name":       "Update Item A",
					"value_type": "3",
					"history":    "90d",
					"trends":     "365d",
					"delay":      "1m",
					"type":       "0", // passive agent
				}),
			},
			{ // every one of them changed, in life
				Config: item(`
	key       = "test.update.item.b"
	name      = "Update Item B"
	valuetype = "float"
	history   = "30d"
	trends    = "180d"
	delay     = "45s"
	active    = true
	tag {
		key   = "tag-b"
		value = "value-b"
	}
	preprocessor {
		type                 = "jsonpath"
		params               = [ "$.b" ]
		error_handler        = "2"
		error_handler_params = "fallback"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverItem, map[string]string{
						"key_":       "test.update.item.b",
						"name":       "Update Item B",
						"value_type": "0",
						"history":    "30d",
						"trends":     "180d",
						"delay":      "45s",
						"type":       "7", // active agent
					}),
					testAccCheckServerElem(addr, serverItem, "tags", "tag", "tag-b", map[string]string{
						"value": "value-b",
					}),
					testAccCheckServerElem(addr, serverItem, "preprocessing", "type", "12", map[string]string{
						"params":               "$.b",
						"error_handler":        "2",
						"error_handler_params": "fallback",
					}),
				),
			},
		},
	})
}

// TestAccUpdateItemInterfaceID owns itemInterfaceSchema.
//
// It needs its own fixture because interfaceid only means anything on a host,
// and only a host with more than one interface of the same type can show the
// attribute being changed rather than merely defaulted. Probed first against
// all four servers: item.create with interfaceid "0" on a host that HAS
// interfaces is rejected on every version ("No interface found." on 6.0, "the
// host interface ID is expected" from 7.0), so "0" is a template-only value
// and the real move is between two ids.
func TestAccUpdateItemInterfaceID(t *testing.T) {
	const addr = "zabbix_item_agent.testitem"

	item := func(port string) string {
		return `
resource "zabbix_hostgroup" "testgrp" {
	name = "test-update-iface-group"
}
resource "zabbix_host" "testhost" {
	host   = "test-update-iface-host"
	groups = [ zabbix_hostgroup.testgrp.id ]
	interface {
		ip   = "127.0.0.1"
		port = 10050
		main = true
	}
	interface {
		ip   = "127.0.0.2"
		port = 10051
		main = false
	}
}
resource "zabbix_item_agent" "testitem" {
	hostid      = zabbix_host.testhost.id
	interfaceid = one([ for i in zabbix_host.testhost.interface : i.id if i.port == ` + port + ` ])
	key         = "test.update.iface"
	name        = "Update Interface Item"
	valuetype   = "unsigned"
}
`
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: item("10050"),
				Check:  resource.TestCheckResourceAttrSet(addr, "interfaceid"),
			},
			{ // moved to the second interface of the same host
				Config:           item("10051"),
				ConfigPlanChecks: expectUpdate(addr),
				Check:            testAccCheckItemInterfacePort(addr, "10051"),
			},
		},
	})
}

// testAccCheckItemInterfacePort re-reads the item's interfaceid from Zabbix
// and then the interface itself, so that the assertion is about which
// interface the server has the item on rather than about an opaque id the
// test would have had to learn from the provider in the first place.
func testAccCheckItemInterfacePort(addr, wantPort string) resource.TestCheckFunc {
	return testAccCheckServerAttrs(addr, func(api *zabbix.API, id string) (map[string]interface{}, error) {
		obj, err := serverItem(api, id)
		if err != nil {
			return nil, err
		}
		ifaceID, _ := serverValue(obj, "interfaceid")
		var ifaces []map[string]interface{}
		err = api.CallWithErrorParse("hostinterface.get", map[string]interface{}{
			"output":       "extend",
			"interfaceids": []string{serverString(ifaceID)},
		}, &ifaces)
		if err != nil {
			return nil, err
		}
		if len(ifaces) != 1 {
			return nil, fmt.Errorf("hostinterface.get returned %d interfaces for %s", len(ifaces), serverString(ifaceID))
		}
		return ifaces[0], nil
	}, map[string]string{"port": wantPort})
}

// TestAccUpdateItemTypeSpecific owns the three single-attribute backend
// fragments: snmp_oid, the calculated item's formula, and a dependent item's
// master_itemid. master_itemid is the interesting one -- it is a reference to
// another object, which is the shape hostid and ruleid have, and unlike them
// Zabbix takes it on update.
func TestAccUpdateItemTypeSpecific(t *testing.T) {
	config := func(oid, formula, master string) string {
		return hcl(t, updateItemTemplateHCL+`
resource "zabbix_item_snmp" "testsnmp" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.snmp"
	name      = "Update SNMP Item"
	valuetype = "unsigned"
	snmp_oid  = "`+oid+`"
}
resource "zabbix_item_trapper" "testmastera" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.master.a"
	name      = "Update Master A"
	valuetype = "unsigned"
}
resource "zabbix_item_trapper" "testmasterb" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.master.b"
	name      = "Update Master B"
	valuetype = "unsigned"
}
resource "zabbix_item_dependent" "testdep" {
	hostid        = zabbix_template.testtmpl.id
	key           = "test.update.dependent"
	name          = "Update Dependent Item"
	valuetype     = "unsigned"
	master_itemid = `+master+`.id
}
resource "zabbix_item_calculated" "testcalc" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.calculated"
	name      = "Update Calculated Item"
	valuetype = "float"
	formula   = "`+formula+`"
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config("1.3.6.1.2.1.1.3.0", "last(//test.update.master.a)", "zabbix_item_trapper.testmastera"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs("zabbix_item_snmp.testsnmp", serverItem, map[string]string{
						"snmp_oid": "1.3.6.1.2.1.1.3.0",
					}),
					resource.TestCheckResourceAttrPair("zabbix_item_dependent.testdep", "master_itemid",
						"zabbix_item_trapper.testmastera", "id"),
				),
			},
			{
				Config: config("1.3.6.1.2.1.1.5.0", "last(//test.update.master.b)", "zabbix_item_trapper.testmasterb"),
				ConfigPlanChecks: expectUpdate(
					"zabbix_item_snmp.testsnmp",
					"zabbix_item_calculated.testcalc",
					"zabbix_item_dependent.testdep",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs("zabbix_item_snmp.testsnmp", serverItem, map[string]string{
						"snmp_oid": "1.3.6.1.2.1.1.5.0",
					}),
					testAccCheckServerAttrs("zabbix_item_calculated.testcalc", serverItem, map[string]string{
						"params": "last(//test.update.master.b)",
					}),
					testAccCheckDependentMaster("zabbix_item_dependent.testdep", "test.update.master.b"),
				),
			},
		},
	})
}

// testAccCheckDependentMaster asserts the master the *server* has for a
// dependent item, by key rather than by id.
func testAccCheckDependentMaster(addr, wantKey string) resource.TestCheckFunc {
	return testAccCheckServerAttrs(addr, func(api *zabbix.API, id string) (map[string]interface{}, error) {
		obj, err := serverItem(api, id)
		if err != nil {
			return nil, err
		}
		master, _ := serverValue(obj, "master_itemid")
		var items []map[string]interface{}
		err = api.CallWithErrorParse("item.get", map[string]interface{}{
			"output":  "extend",
			"itemids": []string{serverString(master)},
		}, &items)
		if err != nil {
			return nil, err
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("item.get returned %d items for master %s", len(items), serverString(master))
		}
		return items[0], nil
	}, map[string]string{"key_": wantKey})
}

// TestAccUpdateItemHttp owns the HTTP fragment in resource_http_common.go,
// shared by zabbix_item_http, zabbix_proto_item_http and zabbix_lld_http.
//
// Two of these are known to be awkward and are why the whole set is changed in
// one step rather than trusted individually: "timeout" takes "" from 7.0 but
// not on 6.0, and "status_codes" and "posts" were both settable but not
// clearable until this release.
func TestAccUpdateItemHttp(t *testing.T) {
	const addr = "zabbix_item_http.testitem"

	item := func(body string) string {
		return hcl(t, updateItemTemplateHCL+`
resource "zabbix_item_http" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.http"
	name      = "Update HTTP Item"
	valuetype = "text"
`+body+`
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: item(`
	url              = "http://example.com/a"
	request_method   = "get"
	post_type        = "raw"
	posts            = "a=1"
	retrieve_mode    = "body"
	auth_type        = "basic"
	username         = "user-a"
	password         = "pass-a"
	proxy            = "http://proxy-a:3128"
	status_codes     = "200"
	timeout          = "3s"
	follow_redirects = true
	verify_host      = true
	verify_peer      = true
	headers = {
		"X-Update" = "a"
	}
`),
				Check: testAccCheckServerAttrs(addr, serverItem, map[string]string{
					"url":              "http://example.com/a",
					"request_method":   "0",
					"post_type":        "0",
					"posts":            "a=1",
					"retrieve_mode":    "0",
					"authtype":         "1",
					"username":         "user-a",
					"http_proxy":       "http://proxy-a:3128",
					"status_codes":     "200",
					"timeout":          "3s",
					"follow_redirects": "1",
					"verify_host":      "1",
					"verify_peer":      "1",
				}),
			},
			{ // all fifteen changed, in life
				Config: item(`
	url              = "http://example.com/b"
	request_method   = "post"
	post_type        = "json"
	posts            = "{\"b\": 2}"
	retrieve_mode    = "headers"
	auth_type        = "ntlm"
	username         = "user-b"
	password         = "pass-b"
	proxy            = "http://proxy-b:3128"
	status_codes     = "201,202"
	timeout          = "17s"
	follow_redirects = false
	verify_host      = false
	verify_peer      = false
	headers = {
		"X-Update" = "b"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckItemHeader(addr, "X-Update", "b"),
					testAccCheckServerAttrs(addr, serverItem, map[string]string{
						"url":              "http://example.com/b",
						"request_method":   "1",
						"post_type":        "2",
						"posts":            `{"b": 2}`,
						"retrieve_mode":    "1",
						"authtype":         "2",
						"username":         "user-b",
						"password":         "pass-b",
						"http_proxy":       "http://proxy-b:3128",
						"status_codes":     "201,202",
						"timeout":          "17s",
						"follow_redirects": "0",
						"verify_host":      "0",
						"verify_peer":      "0",
					}),
				),
			},
		},
	})
}

// testAccCheckItemHeader asserts one HTTP header as the server holds it.
//
// This one goes through ItemsGet rather than raw JSON because "headers" is the
// one item property whose wire shape depends on the version: a name-indexed
// object below 7.0 and an array of {name, value} from 7.0, which the client
// normalises in one place.
func testAccCheckItemHeader(addr, name, want string) resource.TestCheckFunc {
	return testAccCheckServerAttrs(addr, func(api *zabbix.API, id string) (map[string]interface{}, error) {
		res, err := api.ItemsGet(zabbix.Params{"itemids": []string{id}})
		if err != nil {
			return nil, err
		}
		if len(res) != 1 {
			return nil, fmt.Errorf("item.get returned %d items for id %s, want 1", len(res), id)
		}
		out := map[string]interface{}{}
		for k, v := range res[0].Headers {
			out[k] = v
		}
		return out, nil
	}, map[string]string{name: want})
}
