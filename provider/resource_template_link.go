package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/hashicorp/terraform/helper/hashcode"
	zabbix "github.com/tpretz/go-zabbix-api"
)

func hashItem(v interface{}) int {
	m := v.(map[string]interface{})
	return hashcode.String(m["item_id"].(string))
}

func hashTrigger(v interface{}) int {
	m := v.(map[string]interface{})
	return hashcode.String(m["trigger_id"].(string))
}

func hashLLDRule(v interface{}) int {
	m := v.(map[string]interface{})
	return hashcode.String(m["lld_rule_id"].(string))
}

func resourceTemplateLink() *schema.Resource {
	return &schema.Resource{
		Create: resourceTemplateLinkCreate,
		Read:   resourceTemplateLinkRead,
		Update: resourceTemplateLinkUpdate,
		Delete: resourceTemplateLinkDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"template_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Template ID to manage contents for",
			},
			"item": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      hashItem,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"item_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
							Description:  "Item ID belonging to this template",
						},
					},
				},
			},
			"trigger": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      hashTrigger,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"trigger_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
							Description:  "Trigger ID belonging to this template",
						},
					},
				},
			},
			"lld_rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      hashLLDRule,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"lld_rule_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
							Description:  "LLD rule ID belonging to this template",
						},
					},
				},
			},
		},
	}
}

func resourceTemplateLinkCreate(d *schema.ResourceData, m interface{}) error {
	d.SetId(d.Get("template_id").(string))
	return resourceTemplateLinkRead(d, m)
}

func resourceTemplateLinkRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	templateID := d.Id()

	items, err := api.ItemsGet(zabbix.Params{
		"hostids":   templateID,
		"inherited": false,
		"output":    []string{"itemid"},
	})
	if err != nil {
		return fmt.Errorf("zabbix_template_link read items: %v", err)
	}

	triggers, err := api.TriggersGet(zabbix.Params{
		"hostids":   templateID,
		"inherited": false,
		"output":    []string{"triggerid"},
	})
	if err != nil {
		return fmt.Errorf("zabbix_template_link read triggers: %v", err)
	}

	llds, err := api.LLDsGet(zabbix.Params{
		"hostids":   templateID,
		"inherited": false,
		"output":    []string{"itemid"},
	})
	if err != nil {
		return fmt.Errorf("zabbix_template_link read lld rules: %v", err)
	}

	itemSet := schema.NewSet(hashItem, []interface{}{})
	for _, item := range items {
		itemSet.Add(map[string]interface{}{"item_id": item.ItemID})
	}
	d.Set("item", itemSet)

	triggerSet := schema.NewSet(hashTrigger, []interface{}{})
	for _, trigger := range triggers {
		triggerSet.Add(map[string]interface{}{"trigger_id": trigger.TriggerID})
	}
	d.Set("trigger", triggerSet)

	lldSet := schema.NewSet(hashLLDRule, []interface{}{})
	for _, lld := range llds {
		lldSet.Add(map[string]interface{}{"lld_rule_id": lld.ItemID})
	}
	d.Set("lld_rule", lldSet)

	return nil
}

func resourceTemplateLinkUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	if d.HasChange("item") {
		old, new := d.GetChange("item")
		removed := old.(*schema.Set).Difference(new.(*schema.Set))
		var ids []string
		for _, v := range removed.List() {
			ids = append(ids, v.(map[string]interface{})["item_id"].(string))
		}
		if len(ids) > 0 {
			if err := api.ItemsDeleteByIds(ids); err != nil {
				return fmt.Errorf("zabbix_template_link update delete items: %v", err)
			}
		}
	}

	if d.HasChange("trigger") {
		old, new := d.GetChange("trigger")
		removed := old.(*schema.Set).Difference(new.(*schema.Set))
		var ids []string
		for _, v := range removed.List() {
			ids = append(ids, v.(map[string]interface{})["trigger_id"].(string))
		}
		if len(ids) > 0 {
			if err := api.TriggersDeleteByIds(ids); err != nil {
				return fmt.Errorf("zabbix_template_link update delete triggers: %v", err)
			}
		}
	}

	if d.HasChange("lld_rule") {
		old, new := d.GetChange("lld_rule")
		removed := old.(*schema.Set).Difference(new.(*schema.Set))
		var ids []string
		for _, v := range removed.List() {
			ids = append(ids, v.(map[string]interface{})["lld_rule_id"].(string))
		}
		if len(ids) > 0 {
			if err := api.LLDDeleteByIds(ids); err != nil {
				return fmt.Errorf("zabbix_template_link update delete lld rules: %v", err)
			}
		}
	}

	return resourceTemplateLinkRead(d, m)
}

// resourceTemplateLinkDelete is a no-op: when a template is destroyed, Zabbix
// cascades the deletion of its items and triggers automatically.
func resourceTemplateLinkDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
