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
		Type: schema.TypeString,
		//Required:    true,
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
				"interfaceid": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					//ForceNew: true,
				},
				"dns": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					//ForceNew: true,
				},
				"ip": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					//ForceNew: true,
				},
				"main": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  true,
					//ForceNew: true,
				},
				"port": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "10050",
					//ForceNew: true,
				},
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "agent",
					//ForceNew: true,
				},
			},
		},
		//Required: true,
		//ForceNew: true,
	},
	"groups": &schema.Schema{
		Type: schema.TypeSet,
		Elem: &schema.Schema{Type: schema.TypeString},
		//Required: true,
	},
	"templates": &schema.Schema{
		Type:     schema.TypeSet,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	},
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
		case "host", "interfaces", "groups", "templates":
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
	}

	return
}

func resourceHostCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

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
		return err
	}

	item.Interfaces = interfaces

	items := []zabbix.Host{item}

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
		"selectInterfaces": "extend",
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
		"selectInterfaces": "extend",
		"hostids":          d.Id(),
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

	val := [][]interface{}{}
	for i := 0; i < len(host.Interfaces); i++ {
		current := map[string]interface{}{}
		current["interfaceid"] = host.Interfaces[i].InterfaceID
		current["ip"] = host.Interfaces[i].IP
		current["dns"] = host.Interfaces[i].DNS
		current["main"] = host.Interfaces[i].Main == "1"
		current["port"] = host.Interfaces[i].Port
		current["type"] = HOST_IFACE_TYPES_REV[host.Interfaces[i].Type]
		val = append(val, []interface{}{current})
		// prefix := fmt.Sprintf("interfaces.%d.", i)
		// d.Set(prefix+"ip", host.Interfaces[i].IP)
		// d.Set(prefix+"dns", host.Interfaces[i].DNS)
		// d.Set(prefix+"main", host.Interfaces[i].Main == 1)
		// d.Set(prefix+"port", host.Interfaces[i].Port)
		// d.Set(prefix+"type", HOST_IFACE_TYPES_REV[host.Interfaces[i].Type])
	}
	d.Set("interfaces", val)
	log.Debug("got interfaces: %#v", val)

	return nil
}

func resourceHostUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := zabbix.Host{
		HostID: d.Id(),
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
		return err
	}

	item.Interfaces = interfaces

	items := []zabbix.Host{item}

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
