package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// zabbixTemplateGroupMinVersion is the first Zabbix version that has a
// dedicated templategroup API. Before 6.2, templates lived in host groups.
const zabbixTemplateGroupMinVersion = 60200

// resourceTemplategroup terraform resource handler
func resourceTemplategroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceTemplategroupCreate,
		Read:   resourceTemplategroupRead,
		Update: resourceTemplategroupUpdate,
		Delete: resourceTemplategroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Template group name",
				Required:     true,
			},
		},
	}
}

// dataTemplategroup terraform data handler
func dataTemplategroup() *schema.Resource {
	return &schema.Resource{
		Read: dataTemplategroupRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Template group name",
				Required:     true,
			},
		},
	}
}

// resourceTemplategroupCreate terraform templategroup create function
func resourceTemplategroupCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	// Zabbix < 6.2 has no templategroup API, templates use host groups
	if api.Config.Version < zabbixTemplateGroupMinVersion {
		items := []zabbix.HostGroup{{Name: d.Get("name").(string)}}
		if err := api.HostGroupsCreate(items); err != nil {
			return err
		}
		log.Trace("created templategroup (via hostgroup api): %+v", items[0])
		d.SetId(items[0].GroupID)
		return resourceTemplategroupRead(d, m)
	}

	items := []zabbix.TemplateGroup{{Name: d.Get("name").(string)}}
	if err := api.TemplateGroupsCreate(items); err != nil {
		return err
	}

	log.Trace("created templategroup: %+v", items[0])

	d.SetId(items[0].GroupID)

	return resourceTemplategroupRead(d, m)
}

// templategroupRead generic templategroup read function
func templategroupRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	var id, name string

	if api.Config.Version < zabbixTemplateGroupMinVersion {
		groups, err := api.HostGroupsGet(params)
		if err != nil {
			return err
		}
		if len(groups) < 1 {
			d.SetId("")
			return nil
		}
		if len(groups) > 1 {
			return errors.New("multiple templategroups found")
		}
		id, name = groups[0].GroupID, groups[0].Name
	} else {
		groups, err := api.TemplateGroupsGet(params)
		if err != nil {
			return err
		}
		if len(groups) < 1 {
			d.SetId("")
			return nil
		}
		if len(groups) > 1 {
			return errors.New("multiple templategroups found")
		}
		id, name = groups[0].GroupID, groups[0].Name
	}

	log.Debug("Got templategroup: %s (%s)", name, id)

	d.SetId(id)
	d.Set("name", name)

	return nil
}

// dataTemplategroupRead terraform data resource read handler
func dataTemplategroupRead(d *schema.ResourceData, m interface{}) error {
	return templategroupRead(d, m, zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	})
}

// resourceTemplategroupRead terraform resource read handler
func resourceTemplategroupRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of templategroup with id %s", d.Id())

	return templategroupRead(d, m, zabbix.Params{
		"groupids": d.Id(),
	})
}

// resourceTemplategroupUpdate terraform resource update handler
func resourceTemplategroupUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	if api.Config.Version < zabbixTemplateGroupMinVersion {
		items := []zabbix.HostGroup{{
			GroupID: d.Id(),
			Name:    d.Get("name").(string),
		}}
		if err := api.HostGroupsUpdate(items); err != nil {
			return err
		}
		return resourceTemplategroupRead(d, m)
	}

	items := []zabbix.TemplateGroup{{
		GroupID: d.Id(),
		Name:    d.Get("name").(string),
	}}

	if err := api.TemplateGroupsUpdate(items); err != nil {
		return err
	}

	return resourceTemplategroupRead(d, m)
}

// resourceTemplategroupDelete terraform resource delete handler
func resourceTemplategroupDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	if api.Config.Version < zabbixTemplateGroupMinVersion {
		return api.HostGroupsDeleteByIds([]string{d.Id()})
	}

	return api.TemplateGroupsDeleteByIds([]string{d.Id()})
}
