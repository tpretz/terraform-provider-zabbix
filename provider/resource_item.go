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
                        "address": &schema.Schema{
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
