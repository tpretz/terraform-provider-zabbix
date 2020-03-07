package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

var HOST_IFACE_TYPES = map[string]zabbix.InterfaceType{
	"agent": zabbix.Agent,
	"snmp":  zabbix.SNMP,
	"ipmi":  zabbix.IPMI,
	"jmx":   zabbix.JMX,
}
var HOST_IFACE_TYPES_REV = map[zabbix.InterfaceType]string{
	zabbix.Agent: "agent",
	zabbix.SNMP:  "snmp",
	zabbix.IPMI:  "ipmi",
	zabbix.JMX:   "jmx",
}

var hostSchemaBase = map[string]*schema.Schema{
	"name": &schema.Schema{
		Type:        schema.TypeString,
		Required:    false,
		Optional:    true,
		Computed:    true,
		Description: "displayname",
	},
	"host": &schema.Schema{
		Type:        schema.TypeString,
		Description: "host FQDN",
	},
	"enabled": &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
		Default:  true,
	},
	"interfaces": &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
				},
				"dns": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"ip": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"main": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  true,
				},
				"port": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "10050",
				},
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "agent",
				},
			},
		},
	},
	"groups": &schema.Schema{
		Type: schema.TypeSet,
		Elem: &schema.Schema{Type: schema.TypeString},
	},
	"templates": &schema.Schema{
		Type:     schema.TypeSet,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	},
	"macro": macroListSchema,
}

func resourceHost() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostCreate,
		Read:   resourceHostRead,
		Update: resourceHostUpdate,
		Delete: resourceHostDelete,
		Schema: hostResourceSchema(hostSchemaBase),
	}
}

func dataHost() *schema.Resource {
	return &schema.Resource{
		Read:   dataHostRead,
		Schema: hostDataSchema(hostSchemaBase),
	}
}

func hostResourceSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		schema := *v

		// required
		switch k {
		case "host", "interfaces", "groups":
			schema.Required = true
		}

		o[k] = &schema
	}
	return o
}
func hostDataSchema(m map[string]*schema.Schema) (o map[string]*schema.Schema) {
	o = map[string]*schema.Schema{}
	for k, v := range m {
		schema := *v

		// computed
		switch k {
		case "host", "interfaces", "groups", "templates", "macro":
			schema.Computed = true
		}

		// optional
		switch k {
		case "host":
			schema.Optional = true
		}

		o[k] = &schema
	}

	// lookup vars
	o["hostid"] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}

	return o
}

func hostGenerateInterfaces(d *schema.ResourceData) (interfaces zabbix.HostInterfaces, err error) {
	interfaceCount := d.Get("interfaces.#").(int)
	interfaces = make(zabbix.HostInterfaces, interfaceCount)

	for i := 0; i < interfaceCount; i++ {
		prefix := fmt.Sprintf("interfaces.%d.", i)

		ifaceType := d.Get(prefix + "type").(string)

		typeId, ok := HOST_IFACE_TYPES[ifaceType]

		if !ok {
			err = fmt.Errorf("%s isnt valid interface type", ifaceType)
			return
		}

		ip := d.Get(prefix + "ip").(string)
		dns := d.Get(prefix + "dns").(string)

		if ip == "" && dns == "" {
			err = errors.New("interface requires either an IP or DNS entry")
			return
		}

		interfaces[i] = zabbix.HostInterface{
			IP:    ip,
			DNS:   dns,
			Main:  "0",
			Port:  d.Get(prefix + "port").(string),
			Type:  typeId,
			UseIP: "0",
		}

		if ip != "" {
			interfaces[i].UseIP = "1"
		}

		if d.Get(prefix + "main").(bool) {
			interfaces[i].Main = "1"
		}

		// if we have an id (i.e an update)
		if str := d.Get(prefix + "id").(string); str != "" {
			interfaces[i].InterfaceID = str
		}
	}

	return
}

func buildHostObject(d *schema.ResourceData) (*zabbix.Host, error) {
	item := zabbix.Host{
		Host:   d.Get("host").(string),
		Name:   d.Get("name").(string),
		Status: 0,
	}

	if !d.Get("enabled").(bool) {
		item.Status = 1
	}

	item.GroupIds = buildHostGroupIds(d.Get("groups").(*schema.Set))
	item.TemplateIDs = buildTemplateIds(d.Get("templates").(*schema.Set))

	interfaces, err := hostGenerateInterfaces(d)

	if err != nil {
		return nil, err
	}

	item.Interfaces = interfaces
	item.UserMacros = macroGenerate(d)

	return &item, nil
}

func resourceHostCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item, err := buildHostObject(d)

	if err != nil {
		return err
	}

	items := []zabbix.Host{*item}

	err = api.HostsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created host: %+v", items[0])

	d.SetId(items[0].HostID)

	return resourceHostRead(d, m)
}

func dataHostRead(d *schema.ResourceData, m interface{}) error {
	params := zabbix.Params{
		"selectInterfaces":      "extend",
		"selectParentTemplates": "extend",
		"selectGroups":          "extend",
		"selectMacros":          "extend",
	}

	lookups := []string{"host", "hostid", "name"}
	for _, k := range lookups {
		if v, ok := d.GetOk(k); ok {
			if _, ok := params["filter"]; !ok {
				params["filter"] = map[string]interface{}{}
			}
			params["filter"].(map[string]interface{})[k] = v
		}
	}

	if len(params) < 1 {
		return errors.New("no host lookup attribute")
	}
	log.Debug("performing data lookup with params: %#v", params)

	return hostRead(d, m, params)
}

func resourceHostRead(d *schema.ResourceData, m interface{}) error {
	log.Debug("Lookup of hostgroup with id %s", d.Id())

	return hostRead(d, m, zabbix.Params{
		"selectInterfaces":      "extend",
		"selectParentTemplates": "extend",
		"selectGroups":          "extend",
		"selectMacros":          "extend",
		"hostids":               d.Id(),
	})
}

func hostRead(d *schema.ResourceData, m interface{}, params zabbix.Params) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of host with params %#v", params)

	hosts, err := api.HostsGet(params)

	if err != nil {
		return err
	}

	if len(hosts) < 1 {
		return errors.New("no host found")
	}
	if len(hosts) > 1 {
		return errors.New("multiple hosts found")
	}
	host := hosts[0]

	log.Debug("Got host: %+v", host)

	d.SetId(host.HostID)
	d.Set("name", host.Name)
	d.Set("host", host.Host)
	d.Set("enabled", host.Status == 0)

	d.Set("interfaces", flattenHostInterfaces(host))

	templateSet := schema.NewSet(schema.HashString, []interface{}{})
	for _, v := range host.ParentTemplateIDs {
		templateSet.Add(v.TemplateID)
	}
	d.Set("templates", templateSet)

	groupSet := schema.NewSet(schema.HashString, []interface{}{})
	for _, v := range host.GroupIds {
		groupSet.Add(v.GroupID)
	}
	d.Set("groups", groupSet)

	d.Set("macro", flattenMacros(host.UserMacros))

	return nil
}

func flattenHostInterfaces(host zabbix.Host) []interface{} {
	val := make([]interface{}, len(host.Interfaces))
	for i := 0; i < len(host.Interfaces); i++ {
		val[i] = map[string]interface{}{
			"id":   host.Interfaces[i].InterfaceID,
			"ip":   host.Interfaces[i].IP,
			"dns":  host.Interfaces[i].DNS,
			"main": host.Interfaces[i].Main == "1",
			"port": host.Interfaces[i].Port,
			"type": HOST_IFACE_TYPES_REV[host.Interfaces[i].Type],
		}
	}
	return val
}

func resourceHostUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item, err := buildHostObject(d)

	if err != nil {
		return err
	}

	item.HostID = d.Id()

	items := []zabbix.Host{*item}

	err = api.HostsUpdate(items)

	if err != nil {
		return err
	}

	return resourceHostRead(d, m)
}

func resourceHostDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.HostsDeleteByIds([]string{d.Id()})
}
