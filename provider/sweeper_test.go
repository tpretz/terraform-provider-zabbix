package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// TestMain routes `-sweep`/`-sweep-run` invocations to the sweepers
// registered below instead of running the normal test suite. This is the
// recovery path for a run killed mid-flight, where CheckDestroy never gets a
// chance to run and the fixtures' fixed test-* names are left behind to
// collide with the next run.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// testSweepAPI builds a *zabbix.API directly from the environment, the same
// three variables testAccPreCheck requires. Sweepers run outside of the
// schema.Provider lifecycle (there is no configured testAccProvider.Meta()
// to reuse), so this mirrors providerConfigure by hand.
func testSweepAPI() (*zabbix.API, error) {
	url := os.Getenv("ZABBIX_URL")
	if url == "" {
		return nil, fmt.Errorf("ZABBIX_URL must be set to sweep")
	}

	api, err := zabbix.NewAPI(zabbix.Config{Url: url})
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("ZABBIX_TOKEN"); token != "" {
		api.Auth = token
		return api, nil
	}

	user, pass := os.Getenv("ZABBIX_USER"), os.Getenv("ZABBIX_PASS")
	if user == "" || pass == "" {
		return nil, fmt.Errorf("either ZABBIX_TOKEN or ZABBIX_USER+ZABBIX_PASS must be set to sweep")
	}
	if _, err := api.Login(user, pass); err != nil {
		return nil, err
	}
	return api, nil
}

// sweepTestPrefix is the fixture naming convention every acceptance test in
// this package follows (test-group, testgrp, test-template, testtmpl,
// testhost, testtmplgrp, ...). Fixture names are deliberately NOT
// randomised -- each Zabbix version under test has its own database and the
// sweepers need a predictable prefix to find their own mess, so this must
// stay in sync with the fixtures rather than the other way around.
const sweepTestPrefix = "test"

// sweepTestHostAndTemplateIDs returns the ids of every host and template
// (Zabbix templates are, under the hood, hosts with status=3, so both
// object types answer to hostids-style filters) whose technical name starts
// with the fixture prefix. Both host and template ids are needed to sweep
// graphs and items, which can belong to either.
func sweepTestHostAndTemplateIDs(api *zabbix.API) ([]string, error) {
	var ids []string

	hosts, err := api.HostsGet(zabbix.Params{
		"output":      []string{"hostid"},
		"search":      map[string]interface{}{"host": sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing hosts to sweep: %w", err)
	}
	for _, h := range hosts {
		ids = append(ids, h.HostID)
	}

	tmpls, err := api.TemplatesGet(zabbix.Params{
		"output":      []string{"templateid"},
		"search":      map[string]interface{}{"host": sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing templates to sweep: %w", err)
	}
	for _, tm := range tmpls {
		ids = append(ids, tm.TemplateID)
	}

	return ids, nil
}

func init() {
	// Deletion order matters: Zabbix refuses to delete a host/template group
	// that still contains members, and refuses to delete an item that a
	// graph still references. So: graphs, then items, then hosts/templates,
	// then the groups that contained them.
	resource.AddTestSweepers("zabbix_graph", &resource.Sweeper{
		Name: "zabbix_graph",
		F:    sweepGraphs,
	})

	resource.AddTestSweepers("zabbix_item", &resource.Sweeper{
		Name:         "zabbix_item",
		Dependencies: []string{"zabbix_graph"},
		F:            sweepItems,
	})

	resource.AddTestSweepers("zabbix_host", &resource.Sweeper{
		Name:         "zabbix_host",
		Dependencies: []string{"zabbix_item"},
		F:            sweepHosts,
	})

	resource.AddTestSweepers("zabbix_template", &resource.Sweeper{
		Name:         "zabbix_template",
		Dependencies: []string{"zabbix_item"},
		F:            sweepTemplates,
	})

	// A proxy cannot be deleted while a host is still monitored by it, so
	// this has to follow the hosts.
	resource.AddTestSweepers("zabbix_proxy", &resource.Sweeper{
		Name:         "zabbix_proxy",
		Dependencies: []string{"zabbix_host"},
		F:            sweepProxies,
	})

	resource.AddTestSweepers("zabbix_hostgroup", &resource.Sweeper{
		Name:         "zabbix_hostgroup",
		Dependencies: []string{"zabbix_host"},
		F:            sweepHostGroups,
	})

	resource.AddTestSweepers("zabbix_templategroup", &resource.Sweeper{
		Name:         "zabbix_templategroup",
		Dependencies: []string{"zabbix_template"},
		F:            sweepTemplateGroups,
	})
}

func sweepGraphs(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	parents, err := sweepTestHostAndTemplateIDs(api)
	if err != nil {
		return err
	}
	if len(parents) == 0 {
		return nil
	}

	var ids []string

	graphs, err := api.GraphsGet(zabbix.Params{"output": []string{"graphid"}, "hostids": parents})
	if err != nil {
		return fmt.Errorf("listing graphs to sweep: %w", err)
	}
	for _, g := range graphs {
		ids = append(ids, g.GraphID)
	}

	protos, err := api.GraphProtosGet(zabbix.Params{"output": []string{"graphid"}, "hostids": parents})
	if err != nil {
		return fmt.Errorf("listing graph prototypes to sweep: %w", err)
	}
	for _, g := range protos {
		ids = append(ids, g.GraphID)
	}

	if len(ids) == 0 {
		return nil
	}
	if err := api.GraphsDeleteByIds(ids); err != nil {
		return fmt.Errorf("deleting %d graph(s): %w", len(ids), err)
	}
	return nil
}

func sweepItems(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	parents, err := sweepTestHostAndTemplateIDs(api)
	if err != nil {
		return err
	}
	if len(parents) == 0 {
		return nil
	}

	items, err := api.ItemsGet(zabbix.Params{"output": []string{"itemid"}, "hostids": parents})
	if err != nil {
		return fmt.Errorf("listing items to sweep: %w", err)
	}
	if len(items) > 0 {
		ids := make([]string, len(items))
		for i, it := range items {
			ids[i] = it.ItemID
		}
		if err := api.ItemsDeleteByIds(ids); err != nil {
			return fmt.Errorf("deleting %d item(s): %w", len(ids), err)
		}
	}

	protos, err := api.ProtoItemsGet(zabbix.Params{"output": []string{"itemid"}, "hostids": parents})
	if err != nil {
		return fmt.Errorf("listing item prototypes to sweep: %w", err)
	}
	if len(protos) > 0 {
		ids := make([]string, len(protos))
		for i, it := range protos {
			ids[i] = it.ItemID
		}
		if err := api.ProtoItemsDeleteByIds(ids); err != nil {
			return fmt.Errorf("deleting %d item prototype(s): %w", len(ids), err)
		}
	}

	return nil
}

func sweepHosts(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	hosts, err := api.HostsGet(zabbix.Params{
		"output":      []string{"hostid"},
		"search":      map[string]interface{}{"host": sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return fmt.Errorf("listing hosts to sweep: %w", err)
	}
	if len(hosts) == 0 {
		return nil
	}

	ids := make([]string, len(hosts))
	for i, h := range hosts {
		ids[i] = h.HostID
	}
	if err := api.HostsDeleteByIds(ids); err != nil {
		return fmt.Errorf("deleting %d host(s): %w", len(ids), err)
	}
	return nil
}

func sweepTemplates(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	tmpls, err := api.TemplatesGet(zabbix.Params{
		"output":      []string{"templateid"},
		"search":      map[string]interface{}{"host": sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return fmt.Errorf("listing templates to sweep: %w", err)
	}
	if len(tmpls) == 0 {
		return nil
	}

	ids := make([]string, len(tmpls))
	for i, tm := range tmpls {
		ids[i] = tm.TemplateID
	}
	if err := api.TemplatesDeleteByIds(ids); err != nil {
		return fmt.Errorf("deleting %d template(s): %w", len(ids), err)
	}
	return nil
}

func sweepProxies(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	// The proxy's technical name is "host" before 7.0 and "name" from 7.0,
	// and search parameters are passed to the API verbatim.
	proxies, err := api.ProxiesGet(zabbix.Params{
		"output":      []string{"proxyid"},
		"search":      map[string]interface{}{api.ProxyNameProperty(): sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return fmt.Errorf("listing proxies to sweep: %w", err)
	}
	if len(proxies) == 0 {
		return nil
	}

	ids := make([]string, len(proxies))
	for i, p := range proxies {
		ids[i] = p.ProxyID
	}
	if err := api.ProxiesDeleteByIds(ids); err != nil {
		return fmt.Errorf("deleting %d proxy/proxies: %w", len(ids), err)
	}
	return nil
}

func sweepHostGroups(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	groups, err := api.HostGroupsGet(zabbix.Params{
		"output":      []string{"groupid"},
		"search":      map[string]interface{}{"name": sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return fmt.Errorf("listing host groups to sweep: %w", err)
	}
	if len(groups) == 0 {
		return nil
	}

	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.GroupID
	}
	if err := api.HostGroupsDeleteByIds(ids); err != nil {
		return fmt.Errorf("deleting %d host group(s): %w", len(ids), err)
	}
	return nil
}

func sweepTemplateGroups(_ string) error {
	api, err := testSweepAPI()
	if err != nil {
		return err
	}

	// Template groups don't exist before Zabbix 6.2; nothing to sweep there.
	if api.Config.Version < zabbix.V62 {
		return nil
	}

	groups, err := api.TemplateGroupsGet(zabbix.Params{
		"output":      []string{"groupid"},
		"search":      map[string]interface{}{"name": sweepTestPrefix},
		"startSearch": true,
	})
	if err != nil {
		return fmt.Errorf("listing template groups to sweep: %w", err)
	}
	if len(groups) == 0 {
		return nil
	}

	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.GroupID
	}
	if err := api.TemplateGroupsDeleteByIds(ids); err != nil {
		return fmt.Errorf("deleting %d template group(s): %w", len(ids), err)
	}
	return nil
}
