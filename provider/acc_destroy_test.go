package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// testAccCheckDestroyed returns a resource.TestCheckFunc that verifies every
// resource of the given Terraform resource type recorded in the final
// pre-destroy state no longer exists in Zabbix. The SDK calls CheckDestroy
// after the test's own destroy step has already run, but it is passed the
// last state the framework captured before that destroy -- so the ids in
// rs.Primary.ID are exactly the ones that should now be gone.
//
// Zabbix's *.get calls return an empty result for an unknown id rather than
// an error, so "gone" is len(result) == 0, not err != nil. Getting this
// backwards (treating a lookup error as proof of deletion) would make the
// check pass unconditionally, which is worse than not having it -- it would
// look like coverage without providing any.
func testAccCheckDestroyed(resourceType string, exists func(*zabbix.API, string) (bool, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckDestroyed: provider not configured")
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}

			found, err := exists(api, rs.Primary.ID)
			if err != nil {
				return fmt.Errorf("%s %s: error checking it was destroyed: %s", rs.Type, rs.Primary.ID, err)
			}
			if found {
				return fmt.Errorf("%s %s still exists in Zabbix", rs.Type, rs.Primary.ID)
			}
		}
		return nil
	}
}

func testAccHostGroupExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.HostGroupsGet(zabbix.Params{"groupids": []string{id}})
	return len(res) > 0, err
}

func testAccTemplateGroupExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.TemplateGroupsGet(zabbix.Params{"groupids": []string{id}})
	return len(res) > 0, err
}

func testAccProxyExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.ProxiesGet(zabbix.Params{"proxyids": []string{id}})
	return len(res) > 0, err
}

func testAccHostExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.HostsGet(zabbix.Params{"hostids": []string{id}})
	return len(res) > 0, err
}

func testAccTemplateExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.TemplatesGet(zabbix.Params{"templateids": []string{id}})
	return len(res) > 0, err
}

func testAccGraphExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.GraphsGet(zabbix.Params{"graphids": []string{id}})
	return len(res) > 0, err
}

// testAccItemExists backs every zabbix_item_* resource type: they are all
// the same underlying Zabbix "item" object distinguished by the "type"
// field, and item.get takes the same "itemids" filter regardless, so one
// exists check covers all of them.
func testAccItemExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.ItemsGet(zabbix.Params{"itemids": []string{id}})
	return len(res) > 0, err
}

// testAccProtoItemExists is testAccItemExists for the zabbix_proto_item_*
// resources. Item prototypes live in a separate API namespace
// (itemprototype.get, not item.get), so an id that is gone from one is not
// necessarily gone from the other and the two checks cannot be shared.
func testAccProtoItemExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.ProtoItemsGet(zabbix.Params{"itemids": []string{id}})
	return len(res) > 0, err
}

// testAccLLDExists backs every zabbix_lld_* resource: like items, all LLD
// rules are one object type distinguished by "type", and discoveryrule.get
// filters them all by itemids.
func testAccLLDExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.LLDsGet(zabbix.Params{"itemids": []string{id}})
	return len(res) > 0, err
}

func testAccTriggerExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.TriggersGet(zabbix.Params{"triggerids": []string{id}})
	return len(res) > 0, err
}

func testAccProtoTriggerExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.ProtoTriggersGet(zabbix.Params{"triggerids": []string{id}})
	return len(res) > 0, err
}

func testAccProtoGraphExists(api *zabbix.API, id string) (bool, error) {
	res, err := api.GraphProtosGet(zabbix.Params{"graphids": []string{id}})
	return len(res) > 0, err
}

// testAccCheckAllDestroyed is the CheckDestroy attached to every acceptance
// test in this package. Rather than hand-picking which of the object types
// above a given test's config happens to create, it checks all of them --
// testAccCheckDestroyed is a no-op for a type with nothing of that type in
// the final state, so this is always safe and correct to attach, and it
// means a new resource type dropped into an existing test's fixture is
// covered automatically.
func testAccCheckAllDestroyed(s *terraform.State) error {
	return resource.ComposeAggregateTestCheckFunc(
		testAccCheckDestroyed("zabbix_hostgroup", testAccHostGroupExists),
		testAccCheckDestroyed("zabbix_templategroup", testAccTemplateGroupExists),
		testAccCheckDestroyed("zabbix_host", testAccHostExists),
		testAccCheckDestroyed("zabbix_proxy", testAccProxyExists),
		testAccCheckDestroyed("zabbix_template", testAccTemplateExists),
		testAccCheckDestroyed("zabbix_graph", testAccGraphExists),
		testAccCheckDestroyed("zabbix_proto_graph", testAccProtoGraphExists),
		testAccCheckDestroyed("zabbix_trigger", testAccTriggerExists),
		testAccCheckDestroyed("zabbix_proto_trigger", testAccProtoTriggerExists),

		testAccCheckDestroyed("zabbix_item_agent", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_calculated", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_dependent", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_external", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_http", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_internal", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_simple", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_snmp", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_snmptrap", testAccItemExists),
		testAccCheckDestroyed("zabbix_item_trapper", testAccItemExists),

		testAccCheckDestroyed("zabbix_proto_item_agent", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_calculated", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_dependent", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_external", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_http", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_internal", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_simple", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_snmp", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_snmptrap", testAccProtoItemExists),
		testAccCheckDestroyed("zabbix_proto_item_trapper", testAccProtoItemExists),

		testAccCheckDestroyed("zabbix_lld_agent", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_dependent", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_external", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_http", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_internal", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_simple", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_snmp", testAccLLDExists),
		testAccCheckDestroyed("zabbix_lld_trapper", testAccLLDExists),
	)(s)
}
