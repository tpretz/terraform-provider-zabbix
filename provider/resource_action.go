package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var ACTION_EVENTSOURCE_LOOKUP = map[string]string{
	"trigger":          "0",
	"discovery":        "1",
	"autoregistration": "2",
	"internal":         "3",
	"service":          "4",
}
var ACTION_EVENTSOURCE_LOOKUP_REV = map[string]string{
	"0": "trigger",
	"1": "discovery",
	"2": "autoregistration",
	"3": "internal",
	"4": "service",
}
var ACTION_EVENTSOURCE_ARR = []string{"trigger", "discovery", "autoregistration", "internal", "service"}

var ACTION_STATUS_LOOKUP = map[string]string{
	"enabled":  "0",
	"disabled": "1",
}
var ACTION_STATUS_LOOKUP_REV = map[string]string{
	"0": "enabled",
	"1": "disabled",
}
var ACTION_STATUS_ARR = []string{"enabled", "disabled"}

var ACTION_EVALTYPE_LOOKUP = map[string]string{
	"and_or": "0",
	"and":    "1",
	"or":     "2",
	"custom": "3",
}
var ACTION_EVALTYPE_LOOKUP_REV = map[string]string{
	"0": "and_or",
	"1": "and",
	"2": "or",
	"3": "custom",
}
var ACTION_EVALTYPE_ARR = []string{"and_or", "and", "or", "custom"}

func actionOperationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"operationtype": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					Description: "Operation type (numeric string): 0=send message, 1=remote command, 2=add host, 3=remove host, 4=add to host group, 5=remove from host group, 6=link template, 7=unlink template, 8=enable host, 9=disable host, 10=set inventory mode",
				},
				"esc_period": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					// Escalation only applies to trigger and service actions;
					// elsewhere Zabbix reports no value, so this must be
					// computed rather than defaulted to avoid a permanent diff.
					Computed:    true,
					Description: "Escalation step duration (trigger and service actions only)",
				},
				"esc_step_from": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Computed:    true,
					Description: "Escalation step start (trigger and service actions only)",
				},
				"esc_step_to": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Computed:    true,
					Description: "Escalation step end (trigger and service actions only)",
				},
				"evaltype": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Computed:    true,
					Description: "Operation condition evaluation method",
				},
				"opmessage": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"default_msg": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Default:     "1",
								Description: "Use default message (1=yes, 0=no)",
							},
							"subject": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
								Default:  "",
							},
							"message": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
								Default:  "",
							},
							"mediatypeid": &schema.Schema{
								Type:        schema.TypeString,
								Optional:    true,
								Default:     "0",
								Description: "Media type ID (0 for all)",
							},
						},
					},
				},
				"opmessage_grp": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"usrgrpid": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"opmessage_usr": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"userid": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"opcommand": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"scriptid": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"opcommand_hst": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"hostid": &schema.Schema{
								Type:        schema.TypeString,
								Required:    true,
								Description: "Host ID, 0 for current host",
							},
						},
					},
				},
				"opcommand_grp": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"groupid": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"opgroup": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"groupid": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"optemplate": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"templateid": &schema.Schema{
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"opinventory": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"inventory_mode": &schema.Schema{
								Type:        schema.TypeString,
								Required:    true,
								Description: "Inventory mode: 0=manual, 1=automatic",
							},
						},
					},
				},
			},
		},
	}
}

func resourceAction() *schema.Resource {
	return &schema.Resource{
		Create: resourceActionCreate,
		Read:   resourceActionRead,
		Update: resourceActionUpdate,
		Delete: resourceActionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Action name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"eventsource": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Event source: trigger, discovery, autoregistration, internal, service",
				ValidateFunc: validation.StringInSlice(ACTION_EVENTSOURCE_ARR, false),
			},
			"status": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "enabled",
				Description:  "Status: enabled, disabled",
				ValidateFunc: validation.StringInSlice(ACTION_STATUS_ARR, false),
			},
			"esc_period": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				// Only meaningful for trigger and service actions; Zabbix
				// supplies its own default otherwise.
				Computed:    true,
				Description: "Default escalation step duration (trigger and service actions only)",
			},
			"pause_suppressed": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Whether to pause escalation during maintenance (1=yes, 0=no)",
			},
			"notify_if_canceled": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Whether to notify if escalation is canceled (1=yes, 0=no)",
			},
			"filter": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"evaltype": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Filter evaluation method: and_or, and, or, custom",
							ValidateFunc: validation.StringInSlice(ACTION_EVALTYPE_ARR, false),
						},
						"formula": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Custom expression formula (only for evaltype=custom)",
						},
						"conditions": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"conditiontype": &schema.Schema{
										Type:        schema.TypeString,
										Required:    true,
										Description: "Condition type (numeric string)",
									},
									"operator": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Default:  "0",
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value2": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										Default:  "",
									},
									"formulaid": &schema.Schema{
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Arbitrary ID for use in custom formula (only for evaltype=custom)",
									},
								},
							},
						},
					},
				},
			},
			"operations":          actionOperationSchema(),
			"recovery_operations": actionOperationSchema(),
			"update_operations":   actionOperationSchema(),
		},
	}
}

// actionBuildOperations converts the terraform operation blocks into API
// objects. escalation must be false for recovery and update operations: those
// never escalate, and Zabbix rejects esc_period/esc_step_* on them as
// unexpected parameters.
func actionBuildOperations(raw []interface{}, escalation bool) []zabbix.ActionOperation {
	ops := make([]zabbix.ActionOperation, len(raw))
	for i, o := range raw {
		om := o.(map[string]interface{})
		op := zabbix.ActionOperation{
			OperationType: om["operationtype"].(string),
		}
		if escalation {
			op.EscPeriod = om["esc_period"].(string)
			op.EscStepFrom = om["esc_step_from"].(string)
			op.EscStepTo = om["esc_step_to"].(string)
		}
		// evaltype (and the operation conditions it governs) only exist for
		// trigger actions; other event sources reject it.
		if escalation {
			if v, ok := om["evaltype"].(string); ok && v != "" {
				op.EvalType = v
			}
		}

		// opmessage
		if v, ok := om["opmessage"].([]interface{}); ok && len(v) > 0 {
			msg := v[0].(map[string]interface{})
			op.OpMessage = &zabbix.ActionOpMessage{
				DefaultMsg: msg["default_msg"].(string),
				Subject:    msg["subject"].(string),
				Message:    msg["message"].(string),
			}
			// "0" means "all media types", which is Zabbix's own default.
			// Sending it explicitly is rejected by the operation types that
			// notify all involved users (11 and 12).
			if mt := msg["mediatypeid"].(string); mt != "" && mt != "0" {
				op.OpMessage.MediatypeID = mt
			}
		}

		// opmessage_grp
		if v, ok := om["opmessage_grp"].([]interface{}); ok && len(v) > 0 {
			grps := make([]zabbix.ActionOpMessageGrp, len(v))
			for j, g := range v {
				gm := g.(map[string]interface{})
				grps[j] = zabbix.ActionOpMessageGrp{UsrGrpID: gm["usrgrpid"].(string)}
			}
			op.OpMessageGrp = grps
		}

		// opmessage_usr
		if v, ok := om["opmessage_usr"].([]interface{}); ok && len(v) > 0 {
			usrs := make([]zabbix.ActionOpMessageUsr, len(v))
			for j, u := range v {
				um := u.(map[string]interface{})
				usrs[j] = zabbix.ActionOpMessageUsr{UserID: um["userid"].(string)}
			}
			op.OpMessageUsr = usrs
		}

		// opcommand
		if v, ok := om["opcommand"].([]interface{}); ok && len(v) > 0 {
			cmd := v[0].(map[string]interface{})
			op.OpCommand = &zabbix.ActionOpCommand{ScriptID: cmd["scriptid"].(string)}
		}

		// opcommand_hst
		if v, ok := om["opcommand_hst"].([]interface{}); ok && len(v) > 0 {
			hsts := make([]zabbix.ActionOpCommandHst, len(v))
			for j, h := range v {
				hm := h.(map[string]interface{})
				hsts[j] = zabbix.ActionOpCommandHst{HostID: hm["hostid"].(string)}
			}
			op.OpCommandHst = hsts
		}

		// opcommand_grp
		if v, ok := om["opcommand_grp"].([]interface{}); ok && len(v) > 0 {
			grps := make([]zabbix.ActionOpCommandGrp, len(v))
			for j, g := range v {
				gm := g.(map[string]interface{})
				grps[j] = zabbix.ActionOpCommandGrp{GroupID: gm["groupid"].(string)}
			}
			op.OpCommandGrp = grps
		}

		// opgroup
		if v, ok := om["opgroup"].([]interface{}); ok && len(v) > 0 {
			grps := make([]zabbix.ActionOpGroup, len(v))
			for j, g := range v {
				gm := g.(map[string]interface{})
				grps[j] = zabbix.ActionOpGroup{GroupID: gm["groupid"].(string)}
			}
			op.OpGroup = grps
		}

		// optemplate
		if v, ok := om["optemplate"].([]interface{}); ok && len(v) > 0 {
			tmpls := make([]zabbix.ActionOpTemplate, len(v))
			for j, t := range v {
				tm := t.(map[string]interface{})
				tmpls[j] = zabbix.ActionOpTemplate{TemplateID: tm["templateid"].(string)}
			}
			op.OpTemplate = tmpls
		}

		// opinventory
		if v, ok := om["opinventory"].([]interface{}); ok && len(v) > 0 {
			inv := v[0].(map[string]interface{})
			op.OpInventory = &zabbix.ActionOpInventory{InventoryMode: inv["inventory_mode"].(string)}
		}

		ops[i] = op
	}
	return ops
}

func actionFlattenOperations(ops []zabbix.ActionOperation) []map[string]interface{} {
	result := make([]map[string]interface{}, len(ops))
	for i, op := range ops {
		m := map[string]interface{}{
			"operationtype": op.OperationType,
			"esc_period":    op.EscPeriod,
			"esc_step_from": op.EscStepFrom,
			"esc_step_to":   op.EscStepTo,
			"evaltype":      op.EvalType,
		}

		// opmessage
		if op.OpMessage != nil {
			m["opmessage"] = []map[string]interface{}{{
				"default_msg": op.OpMessage.DefaultMsg,
				"subject":     op.OpMessage.Subject,
				"message":     op.OpMessage.Message,
				"mediatypeid": actionNormaliseMediatypeID(op.OpMessage.MediatypeID),
			}}
		} else {
			m["opmessage"] = []map[string]interface{}{}
		}

		// opmessage_grp
		grps := make([]map[string]interface{}, len(op.OpMessageGrp))
		for j, g := range op.OpMessageGrp {
			grps[j] = map[string]interface{}{"usrgrpid": g.UsrGrpID}
		}
		m["opmessage_grp"] = grps

		// opmessage_usr
		usrs := make([]map[string]interface{}, len(op.OpMessageUsr))
		for j, u := range op.OpMessageUsr {
			usrs[j] = map[string]interface{}{"userid": u.UserID}
		}
		m["opmessage_usr"] = usrs

		// opcommand
		if op.OpCommand != nil {
			m["opcommand"] = []map[string]interface{}{{
				"scriptid": op.OpCommand.ScriptID,
			}}
		} else {
			m["opcommand"] = []map[string]interface{}{}
		}

		// opcommand_hst
		hsts := make([]map[string]interface{}, len(op.OpCommandHst))
		for j, h := range op.OpCommandHst {
			hsts[j] = map[string]interface{}{"hostid": h.HostID}
		}
		m["opcommand_hst"] = hsts

		// opcommand_grp
		cmdGrps := make([]map[string]interface{}, len(op.OpCommandGrp))
		for j, g := range op.OpCommandGrp {
			cmdGrps[j] = map[string]interface{}{"groupid": g.GroupID}
		}
		m["opcommand_grp"] = cmdGrps

		// opgroup
		opGrps := make([]map[string]interface{}, len(op.OpGroup))
		for j, g := range op.OpGroup {
			opGrps[j] = map[string]interface{}{"groupid": g.GroupID}
		}
		m["opgroup"] = opGrps

		// optemplate
		tmpls := make([]map[string]interface{}, len(op.OpTemplate))
		for j, t := range op.OpTemplate {
			tmpls[j] = map[string]interface{}{"templateid": t.TemplateID}
		}
		m["optemplate"] = tmpls

		// opinventory
		if op.OpInventory != nil {
			m["opinventory"] = []map[string]interface{}{{
				"inventory_mode": op.OpInventory.InventoryMode,
			}}
		} else {
			m["opinventory"] = []map[string]interface{}{}
		}

		result[i] = m
	}
	return result
}

// actionNormaliseMediatypeID maps an absent mediatypeid to the schema default
// so an unset value does not read back as a diff.
func actionNormaliseMediatypeID(v string) string {
	if v == "" {
		return "0"
	}
	return v
}

func actionBuildObject(d *schema.ResourceData) zabbix.Action {
	eventsource := d.Get("eventsource").(string)

	a := zabbix.Action{
		Name:        d.Get("name").(string),
		EventSource: ACTION_EVENTSOURCE_LOOKUP[eventsource],
		Status:      ACTION_STATUS_LOOKUP[d.Get("status").(string)],
	}

	// Escalations only exist for trigger and service actions. Zabbix rejects
	// esc_period as an unexpected parameter for the other event sources.
	// Escalations only exist for trigger and service actions. Zabbix rejects
	// esc_period as an unexpected parameter for the other event sources, both
	// on the action itself and on its operations.
	escalates := eventsource == "trigger" || eventsource == "service"
	if escalates {
		a.EscPeriod = d.Get("esc_period").(string)
	}

	if v, ok := d.GetOk("pause_suppressed"); ok && eventsource == "trigger" {
		a.PauseSuppressed = v.(string)
	}
	if v, ok := d.GetOk("notify_if_canceled"); ok && eventsource == "trigger" {
		a.NotifyIfCanceled = v.(string)
	}

	// Filter
	if v, ok := d.GetOk("filter"); ok {
		filterList := v.([]interface{})
		if len(filterList) > 0 {
			fm := filterList[0].(map[string]interface{})
			filter := &zabbix.ActionFilter{
				EvalType: ACTION_EVALTYPE_LOOKUP[fm["evaltype"].(string)],
			}
			if formula, ok := fm["formula"].(string); ok && formula != "" {
				filter.Formula = formula
			}

			if condRaw, ok := fm["conditions"].([]interface{}); ok {
				conditions := make([]zabbix.ActionCondition, len(condRaw))
				for i, c := range condRaw {
					cm := c.(map[string]interface{})
					cond := zabbix.ActionCondition{
						ConditionType: cm["conditiontype"].(string),
						Operator:      cm["operator"].(string),
						Value:         cm["value"].(string),
						Value2:        cm["value2"].(string),
					}
					// Only include formulaid when evaltype is custom (3)
					if filter.EvalType == "3" {
						if fid, ok := cm["formulaid"].(string); ok && fid != "" {
							cond.FormulaID = fid
						}
					}
					conditions[i] = cond
				}
				filter.Conditions = conditions
			}
			a.Filter = filter
		}
	}

	// Operations
	if v, ok := d.GetOk("operations"); ok {
		a.Operations = actionBuildOperations(v.([]interface{}), escalates)
	}

	// Recovery operations
	if v, ok := d.GetOk("recovery_operations"); ok {
		a.RecoveryOperations = actionBuildOperations(v.([]interface{}), false)
	}

	// Update operations
	if v, ok := d.GetOk("update_operations"); ok {
		a.UpdateOperations = actionBuildOperations(v.([]interface{}), false)
	}

	return a
}

func resourceActionCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := actionBuildObject(d)
	items := zabbix.Actions{item}

	err := api.ActionsCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created action: %+v", items[0])
	d.SetId(items[0].ActionID)

	return resourceActionRead(d, m)
}

func resourceActionRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of action with id %s", d.Id())

	action, err := api.ActionGetByID(d.Id())
	if err != nil {
		return err
	}
	if action == nil {
		d.SetId("")
		return nil
	}

	d.SetId(action.ActionID)
	d.Set("name", action.Name)
	d.Set("eventsource", ACTION_EVENTSOURCE_LOOKUP_REV[action.EventSource])
	d.Set("status", ACTION_STATUS_LOOKUP_REV[action.Status])
	d.Set("esc_period", action.EscPeriod)
	d.Set("pause_suppressed", action.PauseSuppressed)
	d.Set("notify_if_canceled", action.NotifyIfCanceled)

	// Filter
	if action.Filter != nil {
		filter := map[string]interface{}{
			"evaltype": ACTION_EVALTYPE_LOOKUP_REV[action.Filter.EvalType],
			"formula":  action.Filter.Formula,
		}
		conditions := make([]map[string]interface{}, len(action.Filter.Conditions))
		for i, c := range action.Filter.Conditions {
			cond := map[string]interface{}{
				"conditiontype": c.ConditionType,
				"operator":      c.Operator,
				"value":         c.Value,
				"value2":        c.Value2,
			}
			// Only read formulaid when evaltype is custom
			if action.Filter.EvalType == "3" {
				cond["formulaid"] = c.FormulaID
			} else {
				cond["formulaid"] = ""
			}
			conditions[i] = cond
		}
		filter["conditions"] = conditions
		d.Set("filter", []interface{}{filter})
	}

	// Operations
	d.Set("operations", actionFlattenOperations(action.Operations))
	d.Set("recovery_operations", actionFlattenOperations(action.RecoveryOperations))
	d.Set("update_operations", actionFlattenOperations(action.UpdateOperations))

	return nil
}

func resourceActionUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := actionBuildObject(d)
	item.ActionID = d.Id()
	// eventsource is immutable (ForceNew)
	item.EventSource = ""

	// Operations must be sent without operationid (already excluded in build)

	err := api.ActionsUpdate(zabbix.Actions{item})
	if err != nil {
		return err
	}

	return resourceActionRead(d, m)
}

func resourceActionDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.ActionsDeleteByIds([]string{d.Id()})
}
