package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

// resourceItemAgent terraform resource for agent items
func resourceItemAgent() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemAgentCreate,
		Read:   resourceItemAgentRead,
		Update: resourceItemAgentUpdate,
		Delete: resourceItemDelete,

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema, itemInterfaceSchema),
	}
}

// buildItemAgentObject create struct for agent item create / update
func buildItemAgentObject(d *schema.ResourceData) *zabbix.Item {
	item := zabbix.Item{
		Key:         d.Get("key").(string),
		HostID:      d.Get("hostid").(string),
		Name:        d.Get("name").(string),
		Type:        zabbix.ZabbixAgent,
		ValueType:   ITEM_VALUE_TYPES[d.Get("valuetype").(string)],
		Delay:       d.Get("delay").(string),
		InterfaceID: d.Get("interfaceid").(string),
	}

	item.Preprocessors = itemGeneratePreprocessors(d)

	return &item
}

// resourceItemAgentCreate terraform entity create for zabbix agent items
func resourceItemAgentCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemAgentObject(d)
	items := []zabbix.Item{*item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemAgentRead(d, m)
}

// resourceItemAgentRead terraform entity read for zabbix agent items
func resourceItemAgentRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of item with id %s", d.Id())

	items, err := api.ItemsGet(zabbix.Params{
		"itemids":             []string{d.Id()},
		"selectPreprocessing": "extend",
	})

	if err != nil {
		return err
	}

	if len(items) < 1 {
		return errors.New("no item found")
	}
	if len(items) > 1 {
		return errors.New("multiple items found")
	}
	item := items[0]

	log.Debug("Got item: %+v", item)

	d.SetId(item.ItemID)
	d.Set("hostid", item.HostID)
	d.Set("interfaceid", item.InterfaceID)
	d.Set("key", item.Key)
	d.Set("name", item.Name)
	d.Set("valuetype", ITEM_VALUE_TYPES_REV[item.ValueType])
	d.Set("delay", item.Delay)

	d.Set("preprocessor", flattenItemPreprocessors(item))

	return nil
}

// resourceItemAgentUpdate terraform entity update for zabbix agent items
func resourceItemAgentUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemAgentObject(d)
	item.ItemID = d.Id()

	items := []zabbix.Item{*item}

	err := api.ItemsUpdate(items)

	if err != nil {
		return err
	}

	return resourceItemAgentRead(d, m)
}
