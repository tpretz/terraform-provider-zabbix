package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var SCRIPT_TYPE_LOOKUP = map[string]string{
	"script":  "0",
	"ssh":     "2",
	"telnet":  "3",
	"webhook": "5",
}
var SCRIPT_TYPE_LOOKUP_REV = map[string]string{
	"0": "script",
	"2": "ssh",
	"3": "telnet",
	"5": "webhook",
}
var SCRIPT_TYPE_ARR = []string{"script", "ssh", "telnet", "webhook"}

var SCRIPT_SCOPE_LOOKUP = map[string]string{
	"action_operation":    "1",
	"manual_host_action":  "2",
	"manual_event_action": "4",
}
var SCRIPT_SCOPE_LOOKUP_REV = map[string]string{
	"1": "action_operation",
	"2": "manual_host_action",
	"4": "manual_event_action",
}
var SCRIPT_SCOPE_ARR = []string{"action_operation", "manual_host_action", "manual_event_action"}

var SCRIPT_EXECUTE_ON_LOOKUP = map[string]string{
	"agent":  "0",
	"server": "1",
	"proxy":  "2",
}
var SCRIPT_EXECUTE_ON_LOOKUP_REV = map[string]string{
	"0": "agent",
	"1": "server",
	"2": "proxy",
}
var SCRIPT_EXECUTE_ON_ARR = []string{"agent", "server", "proxy"}

var SCRIPT_HOST_ACCESS_LOOKUP = map[string]string{
	"read":  "2",
	"write": "3",
}
var SCRIPT_HOST_ACCESS_LOOKUP_REV = map[string]string{
	"2": "read",
	"3": "write",
}
var SCRIPT_HOST_ACCESS_ARR = []string{"read", "write"}

func resourceScript() *schema.Resource {
	return &schema.Resource{
		Create: resourceScriptCreate,
		Read:   resourceScriptRead,
		Update: resourceScriptUpdate,
		Delete: resourceScriptDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Script name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"command": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Command to execute or webhook script body",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Script type: script, ssh, telnet, webhook",
				ValidateFunc: validation.StringInSlice(SCRIPT_TYPE_ARR, false),
			},
			"scope": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Script scope: action_operation, manual_host_action, manual_event_action",
				ValidateFunc: validation.StringInSlice(SCRIPT_SCOPE_ARR, false),
			},
			"execute_on": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "Where to execute: agent, server, proxy",
				ValidateFunc: validation.StringInSlice(SCRIPT_EXECUTE_ON_ARR, false),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Script description",
			},
			"confirmation": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Confirmation text for manual scripts",
			},
			"host_access": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "read",
				Description:  "Required host permission: read, write",
				ValidateFunc: validation.StringInSlice(SCRIPT_HOST_ACCESS_ARR, false),
			},
			"groupid": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "0",
				Description: "Host group ID to limit the script to (0 for all groups)",
			},
			"usrgrpid": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "0",
				Description: "User group ID allowed to run the script (0 for all groups)",
			},
			"menu_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Menu path for manual scripts",
			},
			"timeout": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Script execution timeout (webhook type)",
			},
			"parameters": &schema.Schema{
				// A set, not a list: Zabbix does not preserve the order the
				// parameters were sent in, which would otherwise produce a
				// permanent diff.
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Webhook parameters (name/value pairs)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
		},
	}
}

func scriptBuildObject(d *schema.ResourceData) zabbix.Script {
	s := zabbix.Script{
		Name:         d.Get("name").(string),
		Command:      d.Get("command").(string),
		Type:         SCRIPT_TYPE_LOOKUP[d.Get("type").(string)],
		Scope:        SCRIPT_SCOPE_LOOKUP[d.Get("scope").(string)],
		Description:  d.Get("description").(string),
		Confirmation: d.Get("confirmation").(string),
		HostAccess:   SCRIPT_HOST_ACCESS_LOOKUP[d.Get("host_access").(string)],
		GroupID:      d.Get("groupid").(string),
		UsrGrpID:     d.Get("usrgrpid").(string),
		MenuPath:     d.Get("menu_path").(string),
	}

	if v, ok := d.GetOk("execute_on"); ok {
		s.ExecuteOn = SCRIPT_EXECUTE_ON_LOOKUP[v.(string)]
	}
	if v, ok := d.GetOk("timeout"); ok {
		s.Timeout = v.(string)
	}

	// Parameters
	if v, ok := d.GetOk("parameters"); ok {
		paramsList := v.(*schema.Set).List()
		params := make([]zabbix.ScriptParameter, len(paramsList))
		for i, p := range paramsList {
			pm := p.(map[string]interface{})
			params[i] = zabbix.ScriptParameter{
				Name:  pm["name"].(string),
				Value: pm["value"].(string),
			}
		}
		s.Parameters = params
	}

	return s
}

func resourceScriptCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := scriptBuildObject(d)
	items := zabbix.Scripts{item}

	err := api.ScriptsCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created script: %+v", items[0])
	d.SetId(items[0].ScriptID)

	return resourceScriptRead(d, m)
}

func resourceScriptRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of script with id %s", d.Id())

	scripts, err := api.ScriptsGet(zabbix.Params{
		"scriptids": d.Id(),
	})
	if err != nil {
		return err
	}

	if len(scripts) < 1 {
		d.SetId("")
		return nil
	}
	if len(scripts) > 1 {
		return errors.New("multiple scripts found")
	}
	s := scripts[0]

	d.SetId(s.ScriptID)
	d.Set("name", s.Name)
	d.Set("command", s.Command)
	d.Set("type", SCRIPT_TYPE_LOOKUP_REV[s.Type])
	d.Set("scope", SCRIPT_SCOPE_LOOKUP_REV[s.Scope])
	d.Set("execute_on", SCRIPT_EXECUTE_ON_LOOKUP_REV[s.ExecuteOn])
	d.Set("description", s.Description)
	d.Set("confirmation", s.Confirmation)
	d.Set("host_access", SCRIPT_HOST_ACCESS_LOOKUP_REV[s.HostAccess])
	d.Set("groupid", s.GroupID)
	d.Set("usrgrpid", s.UsrGrpID)
	d.Set("menu_path", s.MenuPath)
	d.Set("timeout", s.Timeout)

	// Parameters
	params := make([]map[string]interface{}, len(s.Parameters))
	for i, p := range s.Parameters {
		params[i] = map[string]interface{}{
			"name":  p.Name,
			"value": p.Value,
		}
	}
	d.Set("parameters", params)

	return nil
}

func resourceScriptUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := scriptBuildObject(d)
	item.ScriptID = d.Id()
	// scope is immutable (ForceNew)
	item.Scope = ""

	err := api.ScriptsUpdate(zabbix.Scripts{item})
	if err != nil {
		return err
	}

	return resourceScriptRead(d, m)
}

func resourceScriptDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ScriptsDeleteByIds([]string{d.Id()})
}
