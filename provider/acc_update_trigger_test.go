package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// U1/U2 for triggers. resource_trigger.go declares its attributes once and
// registers them as both zabbix_trigger and zabbix_proto_trigger, so covering
// them here covers the prototype too -- see the pointer-identity grouping in
// TestUpdateCoverageComplete.

// updateTriggerFixtureHCL supplies the template, the item every expression is
// written against, and two further triggers to depend on.
const updateTriggerFixtureHCL = `
resource "zabbix_templategroup" "testtmplgrp" {
	name = "test-update-trigger-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testtmplgrp.id ]
	host   = "test-update-trigger-template"
}
resource "zabbix_item_trapper" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "test.update.trigger"
	name      = "Update Trigger Item"
	valuetype = "float"
}
resource "zabbix_trigger" "testdepa" {
	name       = "Update Trigger Dependency A"
	expression = "last(/test-update-trigger-template/test.update.trigger)=101"
	depends_on = [ zabbix_item_trapper.testitem ]
}
resource "zabbix_trigger" "testdepb" {
	name       = "Update Trigger Dependency B"
	expression = "last(/test-update-trigger-template/test.update.trigger)=102"
	depends_on = [ zabbix_item_trapper.testitem ]
}
`

// TestAccUpdateTrigger changes every attribute a trigger has on a trigger that
// already exists.
//
// recovery_none and recovery_expression are two views of one server property:
// recovery_mode is 0 (the problem expression recovers it), 1 (a separate
// recovery expression) or 2 (never). They need a third step of their own,
// because Zabbix refuses tag correlation on a trigger that never recovers --
// `Incorrect value for field "correlation_mode": unexpected value "1"`, on
// 6.0.48 and 7.4.13 alike -- so recovery_none and correlation_mode cannot both
// be moved to their second value at the same time.
func TestAccUpdateTrigger(t *testing.T) {
	const addr = "zabbix_trigger.testtrigger"

	trigger := func(body string) string {
		return hcl(t, updateTriggerFixtureHCL+`
resource "zabbix_trigger" "testtrigger" {
	depends_on = [ zabbix_item_trapper.testitem ]
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
				Config: trigger(`
	name                = "Update Trigger A"
	event_name          = "Update Event A"
	expression          = "last(/test-update-trigger-template/test.update.trigger)=1"
	recovery_expression = "last(/test-update-trigger-template/test.update.trigger)=0"
	recovery_none       = false
	comments            = "comment a"
	opdata              = "opdata a"
	url                 = "http://example.com/a"
	priority            = "warn"
	enabled             = true
	multiple            = false
	manual_close        = false
	correlation_mode    = "all"
	dependencies        = [ zabbix_trigger.testdepa.id ]
	tag {
		key   = "tag-a"
		value = "value-a"
	}
`),
				Check: testAccCheckServerAttrs(addr, serverTrigger, map[string]string{
					"description":   "Update Trigger A",
					"event_name":    "Update Event A",
					"comments":      "comment a",
					"opdata":        "opdata a",
					"url":           "http://example.com/a",
					"priority":      "2",
					"status":        "0",
					"type":          "0",
					"manual_close":  "0",
					"recovery_mode": "1",
					// correlation_mode "all" is Zabbix's 0
					"correlation_mode": "0",
				}),
			},
			{ // every one of them changed, in life
				Config: trigger(`
	name                = "Update Trigger B"
	event_name          = "Update Event B"
	expression          = "last(/test-update-trigger-template/test.update.trigger)=2"
	recovery_expression = "last(/test-update-trigger-template/test.update.trigger)=3"
	comments            = "comment b"
	opdata              = "opdata b"
	url                 = "http://example.com/b"
	priority            = "disaster"
	enabled             = false
	multiple            = true
	manual_close        = true
	correlation_mode    = "tag"
	correlation_tag     = "corr-b"
	dependencies        = [ zabbix_trigger.testdepb.id ]
	tag {
		key   = "tag-b"
		value = "value-b"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerAttrs(addr, serverTrigger, map[string]string{
						"description":         "Update Trigger B",
						"event_name":          "Update Event B",
						"expression":          "last(/test-update-trigger-template/test.update.trigger)=2",
						"comments":            "comment b",
						"opdata":              "opdata b",
						"url":                 "http://example.com/b",
						"priority":            "5",
						"status":              "1",
						"type":                "1",
						"manual_close":        "1",
						"recovery_mode":       "1",
						"recovery_expression": "last(/test-update-trigger-template/test.update.trigger)=3",
						"correlation_mode":    "1",
						"correlation_tag":     "corr-b",
					}),
					testAccCheckServerElem(addr, serverTrigger, "tags", "tag", "tag-b", map[string]string{
						"value": "value-b",
					}),
					testAccCheckTriggerDependsOn(addr, "zabbix_trigger.testdepb"),
				),
			},
			{ // recovery_none, which needs correlation off the tag mode first
				Config: trigger(`
	name             = "Update Trigger B"
	event_name       = "Update Event B"
	expression       = "last(/test-update-trigger-template/test.update.trigger)=2"
	recovery_none    = true
	comments         = "comment b"
	opdata           = "opdata b"
	url              = "http://example.com/b"
	priority         = "disaster"
	enabled          = false
	multiple         = true
	manual_close     = true
	correlation_mode = "all"
	dependencies     = [ zabbix_trigger.testdepb.id ]
	tag {
		key   = "tag-b"
		value = "value-b"
	}
`),
				ConfigPlanChecks: expectUpdate(addr),
				Check: testAccCheckServerAttrs(addr, serverTrigger, map[string]string{
					"recovery_mode":       "2",
					"recovery_expression": "",
					"correlation_mode":    "0",
					"correlation_tag":     "",
				}),
			},
		},
	})
}

// testAccCheckTriggerDependsOn asserts the dependency the server holds, by the
// id of the trigger it is supposed to point at.
func testAccCheckTriggerDependsOn(addr, depAddr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		depID, err := testAccStateID(s, depAddr)
		if err != nil {
			return err
		}
		return testAccCheckServerElem(addr, serverTrigger, "dependencies", "triggerid", depID, map[string]string{
			"triggerid": depID,
		})(s)
	}
}
