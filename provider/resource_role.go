package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var ROLE_TYPE = map[string]string{
	"user":        "1",
	"admin":       "2",
	"super_admin": "3",
}

var ROLE_TYPE_REV = map[string]string{
	"1": "user",
	"2": "admin",
	"3": "super_admin",
}

var ROLE_TYPE_ARR = []string{"user", "admin", "super_admin"}

// resourceRole terraform resource handler
func resourceRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceRoleCreate,
		Read:   resourceRoleRead,
		Update: resourceRoleUpdate,
		Delete: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the role",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Role type: user, admin, super_admin",
				ValidateFunc: validation.StringInSlice(ROLE_TYPE_ARR, false),
			},
			"readonly": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the role is read-only (builtin roles)",
			},
			"ui": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "UI element access overrides (elements that differ from ui_default_access)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "UI element name, e.g. monitoring.dashboard",
						},
						"status": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Whether access is enabled (true) or denied (false)",
						},
					},
				},
			},
			"ui_default_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Default access to UI elements not listed in ui rules",
			},
			"actions": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Action access overrides (elements that differ from actions_default_access)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Action name, e.g. edit_dashboards",
						},
						"status": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Whether the action is allowed (true) or denied (false)",
						},
					},
				},
			},
			"actions_default_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Default access to actions not listed in actions rules",
			},
			"api_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether API access is enabled for this role",
			},
			"api_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "deny",
				Description:  "API methods list mode: deny (deny-list) or allow (allow-list)",
				ValidateFunc: validation.StringInSlice([]string{"deny", "allow"}, false),
			},
			"api_methods": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of API methods (affected by api_mode)",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// dataRole terraform data handler
func dataRole() *schema.Resource {
	return &schema.Resource{
		Read: dataRoleRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the role to look up",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role type: user, admin, super_admin",
			},
		},
	}
}

func roleBuildObject(d *schema.ResourceData) zabbix.Role {
	role := zabbix.Role{
		Name: d.Get("name").(string),
		Type: ROLE_TYPE[d.Get("type").(string)],
	}

	rules := &zabbix.RoleRule{}

	// UI rules
	if v, ok := d.GetOk("ui"); ok {
		uiList := v.([]interface{})
		rules.UI = make([]zabbix.RoleRuleUIElement, len(uiList))
		for i, item := range uiList {
			m := item.(map[string]interface{})
			status := "0"
			if m["status"].(bool) {
				status = "1"
			}
			rules.UI[i] = zabbix.RoleRuleUIElement{
				Name:   m["name"].(string),
				Status: status,
			}
		}
	}

	if d.Get("ui_default_access").(bool) {
		rules.UIDefaultAccess = "1"
	} else {
		rules.UIDefaultAccess = "0"
	}

	// Actions rules
	if v, ok := d.GetOk("actions"); ok {
		actionsList := v.([]interface{})
		rules.Actions = make([]zabbix.RoleRuleAction, len(actionsList))
		for i, item := range actionsList {
			m := item.(map[string]interface{})
			status := "0"
			if m["status"].(bool) {
				status = "1"
			}
			rules.Actions[i] = zabbix.RoleRuleAction{
				Name:   m["name"].(string),
				Status: status,
			}
		}
	}

	if d.Get("actions_default_access").(bool) {
		rules.ActionsDefaultAcces = "1"
	} else {
		rules.ActionsDefaultAcces = "0"
	}

	// API access
	if d.Get("api_access").(bool) {
		rules.APIAccess = "1"
	} else {
		rules.APIAccess = "0"
	}

	// API mode: deny=0, allow=1
	if d.Get("api_mode").(string) == "allow" {
		rules.APIMode = "1"
	} else {
		rules.APIMode = "0"
	}

	// API methods
	if v, ok := d.GetOk("api_methods"); ok {
		methods := v.([]interface{})
		rules.API = make([]string, len(methods))
		for i, m := range methods {
			rules.API[i] = m.(string)
		}
	} else {
		rules.API = []string{}
	}

	role.Rules = rules
	return role
}

func resourceRoleCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	role := roleBuildObject(d)
	items := zabbix.Roles{role}

	err := api.RolesCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created role: %s", items[0].RoleID)
	d.SetId(items[0].RoleID)

	return resourceRoleRead(d, m)
}

func roleRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	params["selectRules"] = "extend"

	roles, err := api.RolesGet(params)
	if err != nil {
		return err
	}

	if len(roles) < 1 {
		d.SetId("")
		return nil
	}
	if len(roles) > 1 {
		return errors.New("multiple roles found")
	}
	t := roles[0]

	d.SetId(t.RoleID)
	d.Set("name", t.Name)
	d.Set("type", ROLE_TYPE_REV[t.Type])
	d.Set("readonly", t.ReadOnly == "1")

	if t.Rules != nil {
		// UI and Actions: Zabbix always returns ALL elements (not just overrides),
		// so we do NOT read them back - the state preserves what was configured.
		// Only set the scalar rule fields that are readable.

		d.Set("ui_default_access", t.Rules.UIDefaultAccess == "1")
		d.Set("actions_default_access", t.Rules.ActionsDefaultAcces == "1")

		// API
		d.Set("api_access", t.Rules.APIAccess == "1")

		if t.Rules.APIMode == "1" {
			d.Set("api_mode", "allow")
		} else {
			d.Set("api_mode", "deny")
		}

		d.Set("api_methods", t.Rules.API)
	}

	return nil
}

func dataRoleRead(d *schema.ResourceData, m interface{}) error {
	return roleRead(d, m, zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	})
}

func resourceRoleRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of role with id %s", d.Id())
	return roleRead(d, m, zabbix.Params{
		"roleids": d.Id(),
	})
}

func resourceRoleUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	role := roleBuildObject(d)
	role.RoleID = d.Id()

	err := api.RolesUpdate(zabbix.Roles{role})
	if err != nil {
		return err
	}

	return resourceRoleRead(d, m)
}

func resourceRoleDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.RolesDeleteByIds([]string{d.Id()})
}
