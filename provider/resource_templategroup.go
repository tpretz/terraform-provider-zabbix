package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// templateGroupsSupported reports whether the server has a separate
// templategroup API. Zabbix split template groups out of host groups in 6.2;
// below that templates are members of host groups and there is no
// templategroup.* method to call.
func templateGroupsSupported(m interface{}) bool {
	return m.(*zabbix.API).Config.Version >= zabbix.V62
}

// errTemplateGroupUnsupported is returned when zabbix_templategroup is used
// against a server older than 6.2.
func errTemplateGroupUnsupported(m interface{}) error {
	return fmt.Errorf(
		"zabbix_templategroup requires Zabbix 6.2 or later (this server reports %d); "+
			"on 6.0/6.1 templates are members of host groups - use zabbix_hostgroup instead",
		m.(*zabbix.API).Config.Version)
}

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
				Description:  "Template Group Name (Zabbix 6.2 and later)",
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
				Description:  "Template Group Name (Zabbix 6.2 and later)",
				Required:     true,
			},
		},
	}
}

// resourceTemplategroupCreate terraform templategroup create function
func resourceTemplategroupCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	if !templateGroupsSupported(m) {
		return errTemplateGroupUnsupported(m)
	}

	items := zabbix.TemplateGroups{{
		Name: d.Get("name").(string),
	}}

	if err := api.TemplateGroupsCreate(items); err != nil {
		return err
	}

	log.Trace("created templategroup: %+v", items[0])

	d.SetId(items[0].GroupID)

	return resourceTemplategroupRead(d, m)
}

// templategroupRead shared read implementation
func templategroupRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	if !templateGroupsSupported(m) {
		return errTemplateGroupUnsupported(m)
	}

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
	t := groups[0]

	log.Debug("Got templategroup: %+v", t)

	d.SetId(t.GroupID)
	d.Set("name", t.Name)

	return nil
}

// dataTemplategroupRead terraform data resource read handler
func dataTemplategroupRead(d *schema.ResourceData, m interface{}) error {
	err := templategroupRead(d, m, zabbix.Params{
		"filter": map[string]interface{}{
			"name": d.Get("name"),
		},
	})
	if err != nil {
		return err
	}
	return dataSourceFound(d, "templategroup", "name")
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

	if !templateGroupsSupported(m) {
		return errTemplateGroupUnsupported(m)
	}

	items := zabbix.TemplateGroups{{
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

	if !templateGroupsSupported(m) {
		return errTemplateGroupUnsupported(m)
	}

	return api.TemplateGroupsDeleteByIds([]string{d.Id()})
}
