package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

func resourceHostgroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostgroupCreate,
		Read:   resourceHostgroupRead,
		Update: resourceHostgroupUpdate,
		Delete: resourceHostgroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataHostgroup() *schema.Resource {
	return &schema.Resource{
		Read: dataHostgroupRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceHostgroupCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.HostGroup{
		Name: d.Get("name").(string),
	}

	items := []zabbix.HostGroup{item}

	err := api.HostGroupsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created hostgroup: %+v", items[0])

	d.SetId(items[0].GroupID)

	return resourceHostgroupRead(d, m)
}

func hostgroupRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	hostgroups, err := api.HostGroupsGet(params)

	if err != nil {
		return err
	}

	if len(hostgroups) < 1 {
		return errors.New("no hostgroup found")
	}
	if len(hostgroups) > 1 {
		return errors.New("multiple hostgroups found")
	}
	t := hostgroups[0]

	log.Debug("Got hostgroup: %+v", t)

	d.SetId(t.GroupID)
	d.Set("name", t.Name)

	return nil
}

func dataHostgroupRead(d *schema.ResourceData, m interface{}) error {
	return hostgroupRead(d, m, zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	})
}

func resourceHostgroupRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of hostgroup with id %s", d.Id())

	return hostgroupRead(d, m, zabbix.Params{
		"groupids": d.Id(),
	})
}

func resourceHostgroupUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.HostGroup{
		GroupID: d.Id(),
		Name:    d.Get("name").(string),
	}

	items := []zabbix.HostGroup{item}

	err := api.HostGroupsUpdate(items)

	if err != nil {
		return err
	}

	return resourceHostgroupRead(d, m)
}

func resourceHostgroupDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.HostGroupsDeleteByIds([]string{d.Id()})
}
