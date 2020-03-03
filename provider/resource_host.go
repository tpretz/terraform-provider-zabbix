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

func resourceHost() *schema.Resource {
	return &schema.Resource{
		Create: resourceHostCreate,
		Read:   resourceHostRead,
		Update: resourceHostUpdate,
		Delete: resourceHostDelete,

		Schema: map[string]*schema.Schema{
			"hostid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    false,
				Optional:    true,
				Description: "displayname",
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
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
							Required: true,
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
				Required: true,
				//ForceNew: true,
			},
			"groups": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeInt},
				Required: true,
			},
			"templates": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeInt},
				Optional: true,
			},
		},
	}
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
			Main:  0,
			Port:  d.Get(prefix + "port").(string),
			Type:  typeId,
			UseIP: 0,
		}

		if ip != "" {
			interfaces[i].UseIP = 1
		}

		if d.Get(prefix + "main").(bool) {
			interfaces[i].Main = 1
		}
	}

	return
}

func buildHostGroupIds(s *schema.Set) zabbix.HostGroupIDs {
	list := s.List()

	groups := make(zabbix.HostGroupIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.HostGroupID{
			GroupID: list[i].(string),
		}
	}

	return groups
}

func buildTemplateIds(s *schema.Set) zabbix.TemplateIDs {
	list := s.List()

	groups := make(zabbix.TemplateIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.TemplateID{
			TemplateID: list[i].(string),
		}
	}

	return groups
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

	d.Set("hostid", items[0].HostID)
	d.SetId(items[0].HostID)

	return resourceHostRead(d, m)
}

func resourceHostRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	id := d.Get("groupid").(string)

	log.Debug("Lookup of hostgroup with id %s", id)

	host, err := api.HostGetByID(d.Id())

	if err != nil {
		return err
	}

	log.Debug("Got host: %+v", host)

	d.Set("hostid", host.HostID)
	d.Set("name", host.Name)
	d.Set("enabled", host.Status == 0)

	d.Set("interfaces", host.Interfaces)

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
