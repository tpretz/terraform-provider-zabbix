package provider

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Shared helpers for the C1-C7 collection checklist in PLAN.md
// § "The unit of work".
//
// Two things are hard to assert with the stock TestCheckFunc set, and both
// are needed by the checklist:
//
//   - C6 ("remove one, then all") demands that a removal is shown *reaching
//     the server*, not merely leaving Terraform state. State is written by the
//     provider's own read function, so a collection the provider never sent
//     would still look correct in state whenever the read path shares the
//     mistake -- which is exactly the shape of bug C6 exists to catch. The
//     testAccCheck*Count helpers below re-read the object straight from
//     Zabbix, with the same select* parameters the resource read uses, and
//     count what actually came back.
//
//   - C5 ("edit one of many") demands that the edited element kept its
//     server-assigned id rather than being silently deleted and recreated.
//     That needs a value carried from one test step to the next, which
//     testAccRecordSetElemAttr / testAccCheckSetElemAttr do.
//
// Set elements are always located by *content* here, never by index: the
// indices a set has in test state are artefacts of the JSON-state shim.
//
// Where the plural coverage lives
// -------------------------------
//
// Several collections come from one shared file and are merged into many
// resources. Those are tested to full C1-C7 *once*, against one representative
// resource, rather than in eleven near-identical fixtures; the other resources
// keep a single-element smoke check. Read an absence below as that decision,
// not as an oversight.
//
//	collection            code path              plural coverage
//	--------------------  ---------------------  ----------------------------------
//	tag                   common_tag.go          TestAccResourceItemAgentTags
//	                                             (+ zabbix_host and zabbix_trigger /
//	                                             zabbix_proto_trigger, which build
//	                                             and clear tags in their own update
//	                                             paths rather than via common_item)
//	macro                 common_macro.go        TestAccResourceTemplateCollections
//	preprocessor (item)   common_item.go         TestAccResourceItemAgent
//	preprocessor (LLD)    common_lld.go          TestAccResourceLLDPreprocessor
//	                                             -- a separate implementation, so
//	                                             the item test says nothing about it
//	condition             common_lld.go          TestAccResourceLLDTrapper
//	headers               resource_http_common.go TestAccResourceItemHttp
//	item (graph)          resource_graph.go      TestAccResourceGraph,
//	                                             TestAccResourceProtoGraph
//	interface             resource_host.go       TestAccResourceHostMultiInterface
//	groups, templates     resource_host.go,      TestAccResourceHostCollections,
//	                      resource_template.go   TestAccResourceTemplateCollections
//	                                             -- both, because only one of the
//	                                             two update paths used to filter
//	                                             templates_clear
//	dependencies          resource_trigger.go    TestAccResourceTrigger,
//	                                             TestAccResourceProtoTrigger

// testAccStateID returns the primary id recorded in state for addr.
func testAccStateID(s *terraform.State, addr string) (string, error) {
	rs, ok := s.RootModule().Resources[addr]
	if !ok {
		return "", fmt.Errorf("%s not found in state", addr)
	}
	if rs.Primary == nil || rs.Primary.ID == "" {
		return "", fmt.Errorf("%s has no id in state", addr)
	}
	return rs.Primary.ID, nil
}

// testAccCheckServerCount re-reads the object addr points at from Zabbix and
// requires count to report exactly want elements. what names the collection
// in the failure message.
func testAccCheckServerCount(addr, what string, want int, count func(*zabbix.API, string) (int, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckServerCount: provider not configured")
		}

		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}

		got, err := count(api, id)
		if err != nil {
			return fmt.Errorf("%s (%s): reading %s back from the server: %s", addr, id, what, err)
		}
		if got != want {
			return fmt.Errorf("%s (%s): server reports %d %s, want %d", addr, id, got, what, want)
		}
		return nil
	}
}

// --- per-object element counts, straight from the API ----------------------

func testAccCheckGraphItemCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "graph items", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.GraphsGet(zabbix.Params{"graphids": id, "selectGraphItems": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("graph.get returned %d graphs", len(res))
		}
		return len(res[0].GraphItems), nil
	})
}

func testAccCheckProtoGraphItemCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "graph prototype items", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.GraphProtosGet(zabbix.Params{"graphids": id, "selectGraphItems": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("graphprototype.get returned %d graphs", len(res))
		}
		return len(res[0].GraphItems), nil
	})
}

func testAccCheckLLDConditionCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "LLD filter conditions", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.LLDsGet(zabbix.Params{"itemids": []string{id}, "selectFilter": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("discoveryrule.get returned %d rules", len(res))
		}
		return len(res[0].Filter.Conditions), nil
	})
}

func testAccCheckLLDMacroPathCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "LLD macro paths", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.LLDsGet(zabbix.Params{"itemids": []string{id}, "selectLLDMacroPaths": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("discoveryrule.get returned %d rules", len(res))
		}
		if res[0].MacroPaths == nil {
			return 0, nil
		}
		return len(*res[0].MacroPaths), nil
	})
}

func testAccCheckLLDPreprocessorCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "preprocessing steps", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.LLDsGet(zabbix.Params{"itemids": []string{id}, "selectPreprocessing": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("discoveryrule.get returned %d rules", len(res))
		}
		return len(res[0].Preprocessors), nil
	})
}

func testAccCheckItemPreprocessorCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "preprocessing steps", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.ItemsGet(zabbix.Params{"itemids": []string{id}, "selectPreprocessing": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("item.get returned %d items", len(res))
		}
		return len(res[0].Preprocessors), nil
	})
}

func testAccCheckItemTagCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "tags", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.ItemsGet(zabbix.Params{"itemids": []string{id}, "selectTags": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("item.get returned %d items", len(res))
		}
		return len(res[0].Tags), nil
	})
}

func testAccCheckItemHeaderCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "HTTP headers", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.ItemsGet(zabbix.Params{"itemids": []string{id}})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("item.get returned %d items", len(res))
		}
		return len(res[0].Headers), nil
	})
}

func testAccCheckHostTemplateCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "linked templates", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.HostsGet(zabbix.Params{"hostids": id, "selectParentTemplates": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("host.get returned %d hosts", len(res))
		}
		return len(res[0].ParentTemplateIDs), nil
	})
}

func testAccCheckHostGroupCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "host groups", want, func(api *zabbix.API, id string) (int, error) {
		// hostGroupSelectParam picks selectHostGroups on 7.2+ and
		// selectGroups below it; HostsGet folds both into GroupIds.
		res, err := api.HostsGet(zabbix.Params{
			"hostids":                 id,
			hostGroupSelectParam(api): "extend",
		})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("host.get returned %d hosts", len(res))
		}
		return len(res[0].GroupIds), nil
	})
}

func testAccCheckHostMacroCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "macros", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.HostsGet(zabbix.Params{"hostids": id, "selectMacros": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("host.get returned %d hosts", len(res))
		}
		return len(res[0].UserMacros), nil
	})
}

func testAccCheckHostTagCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "tags", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.HostsGet(zabbix.Params{"hostids": id, "selectTags": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("host.get returned %d hosts", len(res))
		}
		return len(res[0].Tags), nil
	})
}

func testAccCheckTemplateLinkedCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "linked templates", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.TemplatesGet(zabbix.Params{"templateids": id, "selectParentTemplates": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("template.get returned %d templates", len(res))
		}
		return len(res[0].ParentTemplates), nil
	})
}

func testAccCheckTemplateGroupCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "template groups", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.TemplatesGet(zabbix.Params{
			"templateids":                 id,
			templateGroupSelectParam(api): "extend",
		})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("template.get returned %d templates", len(res))
		}
		return len(res[0].Groups), nil
	})
}

func testAccCheckTemplateMacroCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "macros", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.TemplatesGet(zabbix.Params{"templateids": id, "selectMacros": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("template.get returned %d templates", len(res))
		}
		return len(res[0].UserMacros), nil
	})
}

func testAccCheckTriggerDependencyCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "trigger dependencies", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.TriggersGet(zabbix.Params{"triggerids": id, "selectDependencies": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("trigger.get returned %d triggers", len(res))
		}
		return len(res[0].Dependencies), nil
	})
}

func testAccCheckTriggerTagCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "tags", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.TriggersGet(zabbix.Params{"triggerids": id, "selectTags": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("trigger.get returned %d triggers", len(res))
		}
		return len(res[0].Tags), nil
	})
}

func testAccCheckProtoTriggerTagCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "tags", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.ProtoTriggersGet(zabbix.Params{"triggerids": id, "selectTags": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("triggerprototype.get returned %d triggers", len(res))
		}
		return len(res[0].Tags), nil
	})
}

func testAccCheckProtoTriggerDependencyCount(addr string, want int) resource.TestCheckFunc {
	return testAccCheckServerCount(addr, "trigger prototype dependencies", want, func(api *zabbix.API, id string) (int, error) {
		res, err := api.ProtoTriggersGet(zabbix.Params{"triggerids": id, "selectDependencies": "extend"})
		if err != nil {
			return 0, err
		}
		if len(res) != 1 {
			return 0, fmt.Errorf("triggerprototype.get returned %d triggers", len(res))
		}
		return len(res[0].Dependencies), nil
	})
}

// --- locating a set element by content, in state ---------------------------

// testAccSetElemAttr returns the value of wantAttr on the single element of
// the set attribute attr where keyAttr == keyVal.
//
// The flat state map indexes set elements by an integer position, but that
// position is assigned by the JSON-state shim and means nothing, so the
// element is found by scanning for its content.
func testAccSetElemAttr(s *terraform.State, addr, attr, keyAttr, keyVal, wantAttr string) (string, error) {
	rs, ok := s.RootModule().Resources[addr]
	if !ok {
		return "", fmt.Errorf("%s not found in state", addr)
	}

	countRaw, ok := rs.Primary.Attributes[attr+".#"]
	if !ok {
		return "", fmt.Errorf("%s has no attribute %q", addr, attr)
	}
	count, err := strconv.Atoi(countRaw)
	if err != nil {
		return "", fmt.Errorf("%s: %s.# is %q, not a number", addr, attr, countRaw)
	}

	for i := 0; i < count; i++ {
		if rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", attr, i, keyAttr)] != keyVal {
			continue
		}
		v, ok := rs.Primary.Attributes[fmt.Sprintf("%s.%d.%s", attr, i, wantAttr)]
		if !ok {
			return "", fmt.Errorf("%s: element of %s with %s=%q has no %q", addr, attr, keyAttr, keyVal, wantAttr)
		}
		return v, nil
	}
	return "", fmt.Errorf("%s: no element of %s has %s=%q", addr, attr, keyAttr, keyVal)
}

// testAccRecordAttr stashes a top-level attribute of addr for a later step to
// compare against with resource.TestCheckResourceAttrPtr. Recording and
// comparing are separate steps because TestCheckResourceAttrPtr dereferences
// at check time: pointed at a variable the same step is filling in, it would
// compare against whatever the variable happened to hold beforehand.
func testAccRecordAttr(addr, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		v, ok := rs.Primary.Attributes[attr]
		if attr == "id" {
			v, ok = rs.Primary.ID, rs.Primary.ID != ""
		}
		if !ok || v == "" {
			return fmt.Errorf("%s has no %s to record", addr, attr)
		}
		*dst = v
		return nil
	}
}

// testAccRecordSetElemAttr stashes one attribute of one set element for a
// later step to compare against -- the C5 "kept its server-assigned id"
// assertion, which needs a value to survive from one apply to the next.
func testAccRecordSetElemAttr(addr, attr, keyAttr, keyVal, wantAttr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := testAccSetElemAttr(s, addr, attr, keyAttr, keyVal, wantAttr)
		if err != nil {
			return err
		}
		if v == "" {
			return fmt.Errorf("%s: %s element %s=%q has an empty %s, nothing to compare later",
				addr, attr, keyAttr, keyVal, wantAttr)
		}
		*dst = v
		return nil
	}
}

// testAccCheckSetElemAttr is the other half of testAccRecordSetElemAttr:
// it requires the element still identified by keyAttr=keyVal to carry the
// recorded value.
func testAccCheckSetElemAttr(addr, attr, keyAttr, keyVal, wantAttr string, want *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := testAccSetElemAttr(s, addr, attr, keyAttr, keyVal, wantAttr)
		if err != nil {
			return err
		}
		if v != *want {
			return fmt.Errorf("%s: %s element %s=%q has %s %q, want %q -- the element was replaced, not edited",
				addr, attr, keyAttr, keyVal, wantAttr, v, *want)
		}
		return nil
	}
}
