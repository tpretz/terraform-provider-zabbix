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

	hostGroups, err := getHostGroups(api, d.Get("groups").(*schema.Set))

	if err != nil {
		return err
	}

	host.GroupIds = hostGroups

	interfaces, err := getInterfaces(d)

	if err != nil {
		return nil, err
	}

	host.Interfaces = interfaces

	templates, err := getTemplates(d, api)

	if err != nil {
		return nil, err
	}

	host.TemplateIds = templates

	items := []zabbix.HostGroup{item}

	err := api.HostGroupsCreate(items)

	if err != nil {
		return err
	}

	log.Trace("created hostgroup: %+v", items[0])

	d.Set("groupid", items[0].GroupID)
	d.SetId(items[0].GroupID)

	return resourceHostRead(d, m)
}

func resourceHostRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	id := d.Get("groupid").(string)

	log.Debug("Lookup of hostgroup with id %s", id)

	hostgroups, err := api.HostGroupsGet(zabbix.Params{
		"groupids": id,
	})

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

	d.Set("groupid", t.GroupID)
	d.Set("name", t.Name)

	return nil
}

func resourceHostUpdate(d *schema.ResourceData, m interface{}) error {
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

	return resourceHostRead(d, m)
}

func resourceHostDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.HostGroupsDeleteByIds([]string{d.Id()})
}
