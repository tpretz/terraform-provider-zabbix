package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceItem() *schema.Resource {
	return &schema.Resource{
		Create: resourceItemCreate,
		Read:   resourceItemRead,
		Update: resourceItemUpdate,
		Delete: resourceItemDelete,

		Schema: map[string]*schema.Schema{
			"itemid": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "Zabbix ID",
			},
			"delay": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Item Delay",
			},
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"interfaceid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Host ID",
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
			},
			"value_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceItemCreate(d *schema.ResourceData, m interface{}) error {
	return resourceItemRead(d, m)
}

func resourceItemRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceItemUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceItemRead(d, m)
}

func resourceItemDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
