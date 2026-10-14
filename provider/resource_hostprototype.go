package provider

import (
	"errors"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

func groupPrototypeHash(i interface{}) int {
	return hashElementExcept(i)
}

var groupPrototypeSetSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Set:      groupPrototypeHash,
	Description: "Dynamic host group name patterns (unordered), e.g. `{ name = \"{#HV.NAME}\" }`. " +
		"Every host this prototype discovers is placed into a group named by substituting that host's LLD macro " +
		"values, in addition to the static groups named in `group`",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Group name pattern, may contain {#MACRO} placeholders from the discovery rule",
			},
		},
	},
}

func buildGroupPrototypes(s *schema.Set) zabbix.HostPrototypeGroupPrototypes {
	list := s.List()
	out := make(zabbix.HostPrototypeGroupPrototypes, len(list))
	for i := 0; i < len(list); i++ {
		current := list[i].(map[string]interface{})
		out[i] = zabbix.HostPrototypeGroupPrototype{
			Name: current["name"].(string),
		}
	}
	return out
}

func flattenGroupPrototypes(list zabbix.HostPrototypeGroupPrototypes) *schema.Set {
	set := schema.NewSet(groupPrototypeHash, []interface{}{})
	for i := 0; i < len(list); i++ {
		set.Add(map[string]interface{}{
			"name": list[i].Name,
		})
	}
	return set
}

var schemaHostPrototype = map[string]*schema.Schema{
	"ruleid": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "ID of the discovery rule this host prototype belongs to. Changing it replaces the prototype",
	},
	"host": &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.StringIsNotWhiteSpace,
		Description:  "Technical name pattern for discovered hosts, may contain {#MACRO} placeholders from the discovery rule, e.g. \"{#VM.NAME}\"",
	},
	"name": &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "Visible name pattern for discovered hosts. Defaults to `host` when empty, same as zabbix_host and zabbix_template",
	},
	"group": &schema.Schema{
		Type:        schema.TypeSet,
		Required:    true,
		MinItems:    1,
		Description: "Host group IDs every discovered host is placed into (unordered). At least one is required -- this is Zabbix's groupLinks",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
	},
	"group_prototype": groupPrototypeSetSchema,
	"template": &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		Description: "Template IDs linked to every discovered host (unordered). Removing one here unlinks and clears it",
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9]+$"), "must be a numeric string"),
		},
	},
	"tag": tagSetSchema,
	"enabled": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Whether discovered hosts are monitored. A disabled host prototype still discovers hosts, but they are created unmonitored",
	},
	"discover": &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Whether this prototype keeps discovering new hosts. Set to false to stop discovering new ones and mark previously discovered hosts as lost, without deleting the prototype itself",
	},
}

func resourceHostPrototype() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Zabbix host prototype: the template an LLD discovery rule uses to create a real " +
			"host for each entity it finds (e.g. one host per discovered virtual machine). Does not yet " +
			"cover inventory_mode, custom interfaces, macros, or proxy/monitored_by assignment -- none of those " +
			"are expressible here.",
		Create:        resourceHostPrototypeCreate,
		Read:          resourceHostPrototypeRead,
		Update:        resourceHostPrototypeUpdate,
		Delete:        resourceHostPrototypeDelete,
		CustomizeDiff: visibleNameCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: schemaHostPrototype,
	}
}

func buildHostPrototypeObject(d *schema.ResourceData) zabbix.HostPrototype {
	item := zabbix.HostPrototype{
		RuleID: d.Get("ruleid").(string),
		Host:   d.Get("host").(string),
		Name:   d.Get("name").(string),
		Status: 0,
	}

	if !d.Get("enabled").(bool) {
		item.Status = 1
	}
	if !d.Get("discover").(bool) {
		item.Discover = 1
	}

	item.GroupLinks = buildHostGroupIds(d.Get("group").(*schema.Set))
	item.GroupPrototypes = buildGroupPrototypes(d.Get("group_prototype").(*schema.Set))
	item.TemplateIDs = buildTemplateIds(d.Get("template").(*schema.Set))
	item.Tags = tagGenerate(d)

	return item
}

func resourceHostPrototypeCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildHostPrototypeObject(d)
	items := zabbix.HostPrototypes{item}

	if err := api.HostPrototypesCreate(items); err != nil {
		return err
	}

	log.Trace("created host prototype: %+v", items[0])

	d.SetId(items[0].HostID)

	return resourceHostPrototypeRead(d, m)
}

func resourceHostPrototypeRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of host prototype with id %s", d.Id())

	items, err := api.HostPrototypesGet(zabbix.Params{
		"hostids":               d.Id(),
		"selectGroupLinks":      "extend",
		"selectGroupPrototypes": "extend",
		"selectTemplates":       []string{"templateid"},
		"selectTags":            "extend",
		"selectDiscoveryRule":   "extend",
	})
	if err != nil {
		return err
	}

	if len(items) < 1 {
		d.SetId("")
		return nil
	}
	if len(items) > 1 {
		return errors.New("multiple host prototypes found")
	}
	t := items[0]

	log.Debug("Got host prototype: %+v", t)

	if t.DiscoveryRule != nil {
		d.Set("ruleid", t.DiscoveryRule.ItemID)
	}
	d.Set("host", t.Host)
	d.Set("name", t.Name)
	d.Set("enabled", t.Status == 0)
	d.Set("discover", t.Discover == 0)
	d.Set("group", flattenHostGroupIds(t.GroupLinks))
	d.Set("group_prototype", flattenGroupPrototypes(t.GroupPrototypes))
	d.Set("template", flattenTemplateIds(t.TemplateIDs))
	d.Set("tag", flattenTags(t.Tags))

	return nil
}

func resourceHostPrototypeUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildHostPrototypeObject(d)
	item.HostID = d.Id()

	if err := api.HostPrototypesUpdate(zabbix.HostPrototypes{item}); err != nil {
		return err
	}

	return resourceHostPrototypeRead(d, m)
}

func resourceHostPrototypeDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.HostPrototypesDeleteByIds([]string{d.Id()})
}
