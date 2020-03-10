package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceItemSimple() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemSimpleCreate,
		Read:   resourceItemSimpleRead,
		Update: resourceItemSimpleUpdate,
		Delete: resourceItemDelete,

		Schema: mergeSchemas(itemCommonSchema, itemDelaySchema),
	}
}

func buildItemSimpleObject(d *schema.ResourceData) *zabbix.Item {
	item := zabbix.Item{
		Key:       d.Get("key").(string),
		HostID:    d.Get("hostid").(string),
		Name:      d.Get("name").(string),
		Type:      zabbix.SimpleCheck,
		ValueType: ITEM_VALUE_TYPES[d.Get("valuetype").(string)],
		Delay:     d.Get("delay").(string),
	}

	item.Preprocessors = itemGeneratePreprocessors(d)

	return &item
}

func resourceItemSimpleCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemSimpleObject(d)
	items := []zabbix.Item{*item}

	err := api.ItemsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("crated item: %+v", items[0])

	d.SetId(items[0].ItemID)

	return resourceItemSimpleRead(d, m)
}

func resourceItemSimpleRead(d *schema.ResourceData, m interface{}) error {
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
	d.Set("key", item.Key)
	d.Set("name", item.Name)
	d.Set("valuetype", ITEM_VALUE_TYPES_REV[item.ValueType])
	d.Set("delay", item.Delay)

	d.Set("preprocessor", flattenItemPreprocessors(item))

	return nil
}

func resourceItemSimpleUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := buildItemSimpleObject(d)
	item.ItemID = d.Id()

	items := []zabbix.Item{*item}

	err := api.ItemsUpdate(items)

	if err != nil {
		return err
	}

	return resourceItemSimpleRead(d, m)
}
