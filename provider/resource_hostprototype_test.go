package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// No dedicated sweeper: sweepHosts already deletes every "test"-prefixed
// host, cascading to its discovery rules and their prototypes.

const hostPrototypeFixtureHCL = `
resource "zabbix_hostgroup" "testhpgrpa" {
	name = "test-hostproto-group-a"
}
resource "zabbix_hostgroup" "testhpgrpb" {
	name = "test-hostproto-group-b"
}
resource "zabbix_templategroup" "testhptmplgrp" {
	name = "test-hostproto-template-group"
}
resource "zabbix_template" "testhptmpla" {
	groups = [ zabbix_templategroup.testhptmplgrp.id ]
	host   = "test-hostproto-template-a"
}
resource "zabbix_template" "testhptmplb" {
	groups = [ zabbix_templategroup.testhptmplgrp.id ]
	host   = "test-hostproto-template-b"
}
resource "zabbix_host" "testhpdiscoveryhost" {
	host   = "test-hostproto-discovery-host"
	groups = [ zabbix_hostgroup.testhpgrpa.id ]
}
resource "zabbix_lld_trapper" "testhplld" {
	hostid = zabbix_host.testhpdiscoveryhost.id
	key    = "hostproto.discovery"
	name   = "Test HostProto Discovery"
	delay  = "0"
}
`

func hostPrototypeHCL(t *testing.T, body string) string {
	return hcl(t, hostPrototypeFixtureHCL+`
resource "zabbix_host_prototype" "testproto" {
	ruleid = zabbix_lld_trapper.testhplld.id
`+body+`
}
`)
}

// testAccCheckHostPrototypeGroupPrototypeCount is C6's server-side half:
// re-reads groupPrototypes straight from hostprototype.get (selectGroupPrototypes).
func testAccCheckHostPrototypeGroupPrototypeCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "group prototypes", want, func(api *zabbix.API, id string) (int, error) {
		items, err := api.HostPrototypesGet(zabbix.Params{"hostids": []string{id}, "selectGroupPrototypes": "extend"})
		if err != nil {
			return 0, err
		}
		if len(items) != 1 {
			return 0, fmt.Errorf("hostprototype.get returned %d objects for id %s, want 1", len(items), id)
		}
		return len(items[0].GroupPrototypes), nil
	})
}

// testAccCheckHostPrototypeGroupCount re-reads groupLinks (hostprototype.get
// selectGroupLinks: https://www.zabbix.com/documentation/current/en/manual/api/reference/hostprototype/get).
func testAccCheckHostPrototypeGroupCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "group links", want, func(api *zabbix.API, id string) (int, error) {
		items, err := api.HostPrototypesGet(zabbix.Params{"hostids": []string{id}, "selectGroupLinks": "extend"})
		if err != nil {
			return 0, err
		}
		if len(items) != 1 {
			return 0, fmt.Errorf("hostprototype.get returned %d objects for id %s, want 1", len(items), id)
		}
		return len(items[0].GroupLinks), nil
	})
}

// testAccCheckHostPrototypeTemplateCount re-reads templates (selectTemplates).
func testAccCheckHostPrototypeTemplateCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "templates", want, func(api *zabbix.API, id string) (int, error) {
		items, err := api.HostPrototypesGet(zabbix.Params{"hostids": []string{id}, "selectTemplates": []string{"templateid"}})
		if err != nil {
			return 0, err
		}
		if len(items) != 1 {
			return 0, fmt.Errorf("hostprototype.get returned %d objects for id %s, want 1", len(items), id)
		}
		return len(items[0].TemplateIDs), nil
	})
}

// testAccCheckHostPrototypeTagCount re-reads tags (selectTags).
func testAccCheckHostPrototypeTagCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "tags", want, func(api *zabbix.API, id string) (int, error) {
		items, err := api.HostPrototypesGet(zabbix.Params{"hostids": []string{id}, "selectTags": "extend"})
		if err != nil {
			return 0, err
		}
		if len(items) != 1 {
			return 0, fmt.Errorf("hostprototype.get returned %d objects for id %s, want 1", len(items), id)
		}
		return len(items[0].Tags), nil
	})
}

func TestAccResourceHostPrototype(t *testing.T) {
	const addr = "zabbix_host_prototype.testproto"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // S9a: the documented minimum -- ruleid, host, group only
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id ]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "host", "Test HostProto {#THING}"),
					resource.TestCheckResourceAttr(addr, "name", "Test HostProto {#THING}"), // R2: derived from host
					resource.TestCheckResourceAttr(addr, "enabled", "true"),
					resource.TestCheckResourceAttr(addr, "discover", "true"),
					resource.TestCheckResourceAttr(addr, "group.#", "1"),
					resource.TestCheckResourceAttr(addr, "template.#", "0"),        // C1
					resource.TestCheckResourceAttr(addr, "group_prototype.#", "0"), // C1
					resource.TestCheckResourceAttr(addr, "tag.#", "0"),
					testAccCheckServerAttrs(addr, serverHostPrototype, map[string]string{
						"host": "Test HostProto {#THING}", "name": "Test HostProto {#THING}",
						"status": "0", "discover": "0",
					}),
				),
			},
			{ // U1/U2: every scalar changed in life, plus C2 for template/group_prototype
				Config: hostPrototypeHCL(t, `
	host     = "Test HostProto Renamed {#THING}"
	name     = "Test HostProto Visible {#THING}"
	enabled  = false
	discover = false
	group    = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]
	template = [ zabbix_template.testhptmpla.id ]

	group_prototype {
		name = "Test HostProto Dynamic {#THING}"
	}

	tag {
		key   = "hostprototag"
		value = "hostprotovalue"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "host", "Test HostProto Renamed {#THING}"),
					resource.TestCheckResourceAttr(addr, "name", "Test HostProto Visible {#THING}"),
					resource.TestCheckResourceAttr(addr, "enabled", "false"),
					resource.TestCheckResourceAttr(addr, "discover", "false"),
					resource.TestCheckResourceAttr(addr, "group.#", "2"),
					resource.TestCheckResourceAttr(addr, "template.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(addr, "template.*", "zabbix_template.testhptmpla", "id"),
					resource.TestCheckResourceAttr(addr, "group_prototype.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "group_prototype.*", map[string]string{
						"name": "Test HostProto Dynamic {#THING}",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "tag.*", map[string]string{
						"key": "hostprototag", "value": "hostprotovalue",
					}),
					testAccCheckServerAttrs(addr, serverHostPrototype, map[string]string{
						"host": "Test HostProto Renamed {#THING}", "name": "Test HostProto Visible {#THING}",
						"status": "1", "discover": "1",
					}),
					testAccCheckHostPrototypeGroupCount(addr, 2),
					testAccCheckHostPrototypeTemplateCount(addr, 1),
					testAccCheckHostPrototypeTagCount(addr, 1),
				),
			},
			{ // C3: three group_prototype entries
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto Renamed {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]

	group_prototype {
		name = "Test HostProto Dynamic A {#THING}"
	}
	group_prototype {
		name = "Test HostProto Dynamic B {#THING}"
	}
	group_prototype {
		name = "Test HostProto Dynamic C {#THING}"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "group_prototype.#", "3"),
					testAccCheckHostPrototypeGroupPrototypeCount(addr, 3),
				),
			},
			{ // C4: same three, different order -- a set, so this must plan empty
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto Renamed {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]

	group_prototype {
		name = "Test HostProto Dynamic C {#THING}"
	}
	group_prototype {
		name = "Test HostProto Dynamic A {#THING}"
	}
	group_prototype {
		name = "Test HostProto Dynamic B {#THING}"
	}
`),
				PlanOnly: true,
			},
			{ // C7: import with every collection at its largest
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // C5: edit one of three, assert the others untouched by content
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto Renamed {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]

	group_prototype {
		name = "Test HostProto Dynamic A {#THING}"
	}
	group_prototype {
		name = "Test HostProto Dynamic B {#THING}"
	}
	group_prototype {
		name = "Test HostProto Dynamic C Edited {#THING}"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "group_prototype.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "group_prototype.*", map[string]string{"name": "Test HostProto Dynamic A {#THING}"}),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "group_prototype.*", map[string]string{"name": "Test HostProto Dynamic B {#THING}"}),
					resource.TestCheckTypeSetElemNestedAttrs(addr, "group_prototype.*", map[string]string{"name": "Test HostProto Dynamic C Edited {#THING}"}),
					testAccCheckHostPrototypeGroupPrototypeCount(addr, 3),
				),
			},
			{ // C6: three down to one
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto Renamed {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]

	group_prototype {
		name = "Test HostProto Dynamic A {#THING}"
	}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "group_prototype.#", "1"),
					testAccCheckHostPrototypeGroupPrototypeCount(addr, 1),
				),
			},
			{ // C6: one down to zero, confirmed on the server -- hostprototype.update
				// replaces groupPrototypes wholesale, so an omitted element is a
				// deletion provided the provider sends the property at all
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto Renamed {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "group_prototype.#", "0"),
					testAccCheckHostPrototypeGroupPrototypeCount(addr, 0),
				),
			},
			{ // C1: and empty is stable
				Config: hostPrototypeHCL(t, `
	host  = "Test HostProto Renamed {#THING}"
	group = [ zabbix_hostgroup.testhpgrpa.id, zabbix_hostgroup.testhpgrpb.id ]
`),
				PlanOnly: true,
			},
		},
	})
}

// TestAccForceNewHostPrototypeRuleid is U3: hostprototype.update rejects
// ruleid outright (unlike item/trigger prototypes, where it is merely
// create-only from a given version).
func TestAccForceNewHostPrototypeRuleid(t *testing.T) {
	var protoID string

	protoOn := func(rule string) string {
		return hcl(t, `
resource "zabbix_hostgroup" "testhpfngrp" {
	name = "test-hostproto-forcenew-group"
}
resource "zabbix_host" "testhpfnhost" {
	host   = "test-hostproto-forcenew-host"
	groups = [ zabbix_hostgroup.testhpfngrp.id ]
}
resource "zabbix_lld_trapper" "testhpfnllda" {
	hostid = zabbix_host.testhpfnhost.id
	key    = "hostproto.forcenew.a"
	name   = "ForceNew HostProto LLD A"
	delay  = "0"
}
resource "zabbix_lld_trapper" "testhpfnlldb" {
	hostid = zabbix_host.testhpfnhost.id
	key    = "hostproto.forcenew.b"
	name   = "ForceNew HostProto LLD B"
	delay  = "0"
}
resource "zabbix_host_prototype" "testfnproto" {
	ruleid = `+rule+`.id
	host   = "ForceNew HostProto {#THING}"
	group  = [ zabbix_hostgroup.testhpfngrp.id ]
}
`)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{
				Config: protoOn("zabbix_lld_trapper.testhpfnllda"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_host_prototype.testfnproto", "ruleid", "zabbix_lld_trapper.testhpfnllda", "id"),
					testAccRecordAttr("zabbix_host_prototype.testfnproto", "id", &protoID),
				),
			},
			{ // re-parent onto the other discovery rule
				Config: protoOn("zabbix_lld_trapper.testhpfnlldb"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("zabbix_host_prototype.testfnproto", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("zabbix_host_prototype.testfnproto", "ruleid", "zabbix_lld_trapper.testhpfnlldb", "id"),
					testAccCheckAttrChanged("zabbix_host_prototype.testfnproto", "id", &protoID),
					testAccCheckGone(&protoID, testAccHostPrototypeExists),
				),
			},
		},
	})
}

// TestAccRemoveHostPrototypeDefaults is R1 for enabled and discover: deleting
// the line must revert to the documented default, on the server.
func TestAccRemoveHostPrototypeDefaults(t *testing.T) {
	const addr = "zabbix_host_prototype.testremproto"

	proto := func(body string) string {
		return hcl(t, `
resource "zabbix_hostgroup" "testhpremgrp" {
	name = "test-hostproto-removal-group"
}
resource "zabbix_host" "testhpremhost" {
	host   = "test-hostproto-removal-host"
	groups = [ zabbix_hostgroup.testhpremgrp.id ]
}
resource "zabbix_lld_trapper" "testhpremlld" {
	hostid = zabbix_host.testhpremhost.id
	key    = "hostproto.removal"
	name   = "Removal HostProto LLD"
	delay  = "0"
}
resource "zabbix_host_prototype" "testremproto" {
	ruleid = zabbix_lld_trapper.testhpremlld.id
	host   = "Removal HostProto {#THING}"
	group  = [ zabbix_hostgroup.testhpremgrp.id ]
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
				Config: proto(`
	enabled  = false
	discover = false
`),
				Check: testAccCheckServerAttrs(addr, serverHostPrototype, map[string]string{
					"status": "1", "discover": "1",
				}),
			},
			{ // both lines deleted: back to the documented defaults
				Config:           proto(``),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "enabled", "true"),
					resource.TestCheckResourceAttr(addr, "discover", "true"),
					testAccCheckServerAttrs(addr, serverHostPrototype, map[string]string{
						"status": "0", "discover": "0",
					}),
				),
			},
		},
	})
}
