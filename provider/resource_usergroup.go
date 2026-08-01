package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var USERGROUP_GUI_ACCESS = map[string]string{
	"default":  "0",
	"internal": "1",
	"LDAP":     "2",
	"disable":  "3",
}

var USERGROUP_GUI_ACCESS_REV = map[string]string{
	"0": "default",
	"1": "internal",
	"2": "LDAP",
	"3": "disable",
}

var USERGROUP_GUI_ACCESS_ARR = []string{"default", "internal", "LDAP", "disable"}

var USERGROUP_PERMISSION = map[string]string{
	"deny":       "0",
	"read-only":  "2",
	"read-write": "3",
}

var USERGROUP_PERMISSION_REV = map[string]string{
	"0": "deny",
	"2": "read-only",
	"3": "read-write",
}

var USERGROUP_PERMISSION_ARR = []string{"deny", "read-only", "read-write"}

// resourceUsergroup terraform resource handler
func resourceUsergroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceUsergroupCreate,
		Read:   resourceUsergroupRead,
		Update: resourceUsergroupUpdate,
		Delete: resourceUsergroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the user group",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"gui_access": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "default",
				Description:  "Frontend authentication method: default, internal, LDAP, disable",
				ValidateFunc: validation.StringInSlice(USERGROUP_GUI_ACCESS_ARR, false),
			},
			"users_status": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the user group is enabled (true) or disabled (false)",
			},
			"debug_mode": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether debug mode is enabled for the group",
			},
			"hostgroup_rights": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Host group permissions",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the host group",
						},
						"permission": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Access level: deny, read-only, read-write",
							ValidateFunc: validation.StringInSlice(USERGROUP_PERMISSION_ARR, false),
						},
					},
				},
			},
			"templategroup_rights": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Template group permissions",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the template group",
						},
						"permission": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Access level: deny, read-only, read-write",
							ValidateFunc: validation.StringInSlice(USERGROUP_PERMISSION_ARR, false),
						},
					},
				},
			},
			"tag_filters": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Tag-based permissions",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"groupid": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the host group",
						},
						"tag": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Tag name",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Tag value",
						},
					},
				},
			},
		},
	}
}

// dataUsergroup terraform data handler
func dataUsergroup() *schema.Resource {
	return &schema.Resource{
		Read: dataUsergroupRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the user group",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func usergroupBuildObject(d *schema.ResourceData) zabbix.UserGroup {
	ug := zabbix.UserGroup{
		Name:      d.Get("name").(string),
		GuiAccess: USERGROUP_GUI_ACCESS[d.Get("gui_access").(string)],
	}

	// users_status: true=enabled(0), false=disabled(1)
	if d.Get("users_status").(bool) {
		ug.UsersStatus = "0"
	} else {
		ug.UsersStatus = "1"
	}

	// debug_mode: true=enabled(1), false=disabled(0)
	if d.Get("debug_mode").(bool) {
		ug.DebugMode = "1"
	} else {
		ug.DebugMode = "0"
	}

	// hostgroup_rights
	if v, ok := d.GetOk("hostgroup_rights"); ok {
		rights := v.([]interface{})
		ug.HostGroupRights = make([]zabbix.Permission, len(rights))
		for i, r := range rights {
			rm := r.(map[string]interface{})
			ug.HostGroupRights[i] = zabbix.Permission{
				ID:         rm["id"].(string),
				Permission: USERGROUP_PERMISSION[rm["permission"].(string)],
			}
		}
	}

	// templategroup_rights
	if v, ok := d.GetOk("templategroup_rights"); ok {
		rights := v.([]interface{})
		ug.TemplateGroupRights = make([]zabbix.Permission, len(rights))
		for i, r := range rights {
			rm := r.(map[string]interface{})
			ug.TemplateGroupRights[i] = zabbix.Permission{
				ID:         rm["id"].(string),
				Permission: USERGROUP_PERMISSION[rm["permission"].(string)],
			}
		}
	}

	// tag_filters
	if v, ok := d.GetOk("tag_filters"); ok {
		filters := v.([]interface{})
		ug.TagFilters = make([]zabbix.TagFilter, len(filters))
		for i, f := range filters {
			fm := f.(map[string]interface{})
			ug.TagFilters[i] = zabbix.TagFilter{
				GroupID: fm["groupid"].(string),
				Tag:     fm["tag"].(string),
				Value:   fm["value"].(string),
			}
		}
	}

	return ug
}

func resourceUsergroupCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	ug := usergroupBuildObject(d)
	items := zabbix.UserGroups{ug}

	err := api.UserGroupsCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created usergroup: %s", items[0].UserGroupID)
	d.SetId(items[0].UserGroupID)

	return resourceUsergroupRead(d, m)
}

func usergroupRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	params["selectHostGroupRights"] = "extend"
	params["selectTemplateGroupRights"] = "extend"
	params["selectTagFilters"] = "extend"

	groups, err := api.UserGroupsGet(params)
	if err != nil {
		return err
	}

	if len(groups) < 1 {
		d.SetId("")
		return nil
	}
	if len(groups) > 1 {
		return errors.New("multiple user groups found")
	}
	t := groups[0]

	d.SetId(t.UserGroupID)
	d.Set("name", t.Name)
	d.Set("gui_access", USERGROUP_GUI_ACCESS_REV[t.GuiAccess])
	d.Set("users_status", t.UsersStatus == "0")
	d.Set("debug_mode", t.DebugMode == "1")

	// hostgroup_rights
	hgRights := make([]map[string]interface{}, len(t.HostGroupRights))
	for i, r := range t.HostGroupRights {
		hgRights[i] = map[string]interface{}{
			"id":         r.ID,
			"permission": USERGROUP_PERMISSION_REV[r.Permission],
		}
	}
	d.Set("hostgroup_rights", hgRights)

	// templategroup_rights
	tgRights := make([]map[string]interface{}, len(t.TemplateGroupRights))
	for i, r := range t.TemplateGroupRights {
		tgRights[i] = map[string]interface{}{
			"id":         r.ID,
			"permission": USERGROUP_PERMISSION_REV[r.Permission],
		}
	}
	d.Set("templategroup_rights", tgRights)

	// tag_filters
	tagFilters := make([]map[string]interface{}, len(t.TagFilters))
	for i, f := range t.TagFilters {
		tagFilters[i] = map[string]interface{}{
			"groupid": f.GroupID,
			"tag":     f.Tag,
			"value":   f.Value,
		}
	}
	d.Set("tag_filters", tagFilters)

	return nil
}

func dataUsergroupRead(d *schema.ResourceData, m interface{}) error {
	return usergroupRead(d, m, zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	})
}

func resourceUsergroupRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of usergroup with id %s", d.Id())
	return usergroupRead(d, m, zabbix.Params{
		"usrgrpids": d.Id(),
	})
}

func resourceUsergroupUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	ug := usergroupBuildObject(d)
	ug.UserGroupID = d.Id()

	err := api.UserGroupsUpdate(zabbix.UserGroups{ug})
	if err != nil {
		return err
	}

	return resourceUsergroupRead(d, m)
}

func resourceUsergroupDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.UserGroupsDeleteByIds([]string{d.Id()})
}
