package provider

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// resourceTemplateV0 is the zabbix_template schema as shipped through
// v0.17.0. It exists solely to give the state upgrader a cty type to decode
// prior state with; it is never registered with the provider.
func resourceTemplateV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"groups": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
				},
				Required:    true,
				Description: "Host Group IDs",
			},
			"host": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"templates": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
				},
			},
			"macro": macroSetSchema,
		},
	}
}

// resourceTemplateStateUpgradeV0 handles the host-group -> template-group
// transition in zabbix_template.groups.
//
// Zabbix 6.2 split template groups out of host groups. A state file written by
// provider v0.17.0 therefore records *host* group ids in `groups`, which
// template.create/update on a 6.2+ server rejects.
//
// This upgrader deliberately does NOT rewrite the ids. Zabbix's own 6.2
// database upgrade converted a host group in place (keeping its id) only when
// the group contained templates and no hosts; a mixed group was split into a
// host group and a *new* template group with a fresh id, and an operator may
// since have renamed or merged either side. Nothing in the state file
// distinguishes those cases, so any mechanical translation would be a guess -
// and silently pointing a template at the wrong group is far worse than
// refusing to proceed.
//
// Instead it verifies: ids that are already valid template groups (the
// converted-in-place case, and every re-run after a manual fix) pass through
// untouched; anything else fails with an error naming the offending ids and
// what to do about them.
func resourceTemplateStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	if rawState == nil {
		return rawState, nil
	}

	api, ok := meta.(*zabbix.API)
	if !ok || api == nil {
		// The provider is not configured for this call - `terraform state`
		// subcommands do this. Without a server we cannot tell a good id from
		// a bad one, and guessing is the thing this upgrader exists to avoid.
		log.Warn("zabbix_template state upgrade v0->v1: provider not configured, leaving groups unverified")
		return rawState, nil
	}

	if api.Config.Version < zabbix.V62 {
		// Below 6.2 templates still live in host groups: v0 state is correct
		// as written and there is nothing to migrate.
		log.Debug("zabbix_template state upgrade v0->v1: server is pre-6.2, groups unchanged")
		return rawState, nil
	}

	ids := stateStringList(rawState["groups"])
	if len(ids) == 0 {
		return rawState, nil
	}

	groups, err := api.TemplateGroupsGet(zabbix.Params{
		"groupids": ids,
		"output":   []string{"groupid"},
	})
	if err != nil {
		return nil, fmt.Errorf("zabbix_template state upgrade v0->v1: could not check group ids %s against templategroup.get: %w", strings.Join(ids, ", "), err)
	}

	known := map[string]bool{}
	for _, g := range groups {
		known[g.GroupID] = true
	}

	var missing []string
	for _, id := range ids {
		if !known[id] {
			missing = append(missing, id)
		}
	}

	if len(missing) == 0 {
		log.Debug("zabbix_template state upgrade v0->v1: all %d group ids are template groups, no change needed", len(ids))
		return rawState, nil
	}

	sort.Strings(missing)
	return nil, fmt.Errorf("zabbix_template %s: %s",
		templateStateName(rawState), templateGroupMigrationMessage(api, missing))
}

// templateStateName describes the template being upgraded well enough for a
// user to find it, without assuming any particular attribute is populated.
func templateStateName(rawState map[string]interface{}) string {
	if v, ok := rawState["host"].(string); ok && v != "" {
		return fmt.Sprintf("%q", v)
	}
	if v, ok := rawState["id"].(string); ok && v != "" {
		return "id " + v
	}
	return "(unnamed)"
}

// templateGroupMigrationMessage builds the actionable half of the upgrade
// failure. Where a stale id still resolves to a host group its name is
// included, because the matching template group is usually the one with the
// same name.
func templateGroupMigrationMessage(api *zabbix.API, missing []string) string {
	names := map[string]string{}
	if hgs, err := api.HostGroupsGet(zabbix.Params{
		"groupids": missing,
		"output":   []string{"groupid", "name"},
	}); err == nil {
		for _, g := range hgs {
			names[g.GroupID] = g.Name
		}
	}

	described := make([]string, 0, len(missing))
	for _, id := range missing {
		if n, ok := names[id]; ok {
			described = append(described, fmt.Sprintf("%s (host group %q)", id, n))
		} else {
			described = append(described, id)
		}
	}

	return fmt.Sprintf(
		"`groups` holds ids that are not template groups on this server: %s.\n\n"+
			"Zabbix 6.2 split template groups out of host groups, and this provider "+
			"now requires template group ids here. The old ids cannot be translated "+
			"automatically: Zabbix's 6.2 upgrade kept the id only for groups that "+
			"contained templates and no hosts, and allocated a new id otherwise, "+
			"so rewriting them would be guesswork.\n\n"+
			"To migrate:\n"+
			"  1. Replace the zabbix_hostgroup references in this template's `groups` "+
			"with zabbix_templategroup resources or data sources, e.g.\n"+
			"       data \"zabbix_templategroup\" \"tmpl\" { name = \"Templates/Applications\" }\n"+
			"       resource \"zabbix_template\" \"example\" { groups = [ data.zabbix_templategroup.tmpl.id ] }\n"+
			"  2. Correct the ids already in state with `terraform state rm` followed by "+
			"`terraform import zabbix_template.<name> <templateid>`, or edit them by hand.\n"+
			"See MIGRATING.md for the full v0.17.0 -> v2.0.0 upgrade.",
		strings.Join(described, ", "))
}

// stateStringList coerces a raw state value (JSON-decoded, so []interface{})
// into a list of strings, skipping anything that is not one.
func stateStringList(raw interface{}) []string {
	list, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
