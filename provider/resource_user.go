package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// resourceUser terraform resource handler
func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserCreate,
		Read:   resourceUserRead,
		Update: resourceUserUpdate,
		Delete: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"username": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "User alias used to log in",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "User first name",
			},
			"surname": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "User last name",
			},
			"passwd": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "User password. Write-only, never read back from the API.",
			},
			"roleid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the role assigned to the user",
			},
			"usrgrps": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "IDs of the user groups the user belongs to (at least one required)",
				MinItems:    1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"lang": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Language code, e.g. en_US, or 'default'",
			},
			"timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User timezone, e.g. Europe/London, or 'default'",
			},
			"theme": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User theme: default, blue-theme, dark-theme",
			},
			"autologin": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to enable auto-login",
			},
			"autologout": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "User session lifetime (e.g. 300, 15m). 0 to disable.",
			},
			"refresh": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Automatic refresh period (e.g. 30s)",
			},
			"rows_per_page": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Number of rows per page in web interface tables",
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "URL of the page to redirect the user to after logging in",
			},
			"medias": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "User media (notification) entries",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mediatypeid": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the media type",
						},
						"sendto": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Addresses to send notifications to",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"active": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "Whether the media is enabled",
						},
						"severity": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     63,
							Description: "Trigger severities bitmask to send notifications for (0-63)",
						},
						"period": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "1-7,00:00-24:00",
							Description: "Time period when notifications can be sent",
						},
					},
				},
			},
		},
	}
}

func userBuildObject(d *schema.ResourceData) zabbix.User {
	u := zabbix.User{
		Username: d.Get("username").(string),
		Name:     d.Get("name").(string),
		Surname:  d.Get("surname").(string),
		RoleID:   d.Get("roleid").(string),
		URL:      d.Get("url").(string),
	}

	if v, ok := d.GetOk("passwd"); ok {
		u.Passwd = v.(string)
	}

	// autologin
	if d.Get("autologin").(bool) {
		u.Autologin = "1"
	} else {
		u.Autologin = "0"
	}

	if v, ok := d.GetOk("autologout"); ok {
		u.Autologout = v.(string)
	}
	if v, ok := d.GetOk("refresh"); ok {
		u.Refresh = v.(string)
	}
	if v, ok := d.GetOk("rows_per_page"); ok {
		u.RowsPerPage = v.(string)
	}
	if v, ok := d.GetOk("lang"); ok {
		u.Lang = v.(string)
	}
	if v, ok := d.GetOk("timezone"); ok {
		u.Timezone = v.(string)
	}
	if v, ok := d.GetOk("theme"); ok {
		u.Theme = v.(string)
	}

	// usrgrps
	grpSet := d.Get("usrgrps").(*schema.Set).List()
	u.UserGroups = make([]zabbix.UserGrpID, len(grpSet))
	for i, g := range grpSet {
		u.UserGroups[i] = zabbix.UserGrpID{UserGroupID: g.(string)}
	}

	// medias
	if v, ok := d.GetOk("medias"); ok {
		mediaList := v.([]interface{})
		u.Medias = make([]zabbix.UserMedia, len(mediaList))
		for i, item := range mediaList {
			m := item.(map[string]interface{})
			sendtoRaw := m["sendto"].([]interface{})
			sendto := make([]string, len(sendtoRaw))
			for j, s := range sendtoRaw {
				sendto[j] = s.(string)
			}
			active := "0" // 0 = enabled in Zabbix API
			if !m["active"].(bool) {
				active = "1" // 1 = disabled in Zabbix API
			}
			u.Medias[i] = zabbix.UserMedia{
				MediaTypeID: m["mediatypeid"].(string),
				SendTo:      sendto,
				Active:      active,
				Severity:    userIntToStr(m["severity"].(int)),
				Period:      m["period"].(string),
			}
		}
	}

	return u
}

func userIntToStr(i int) string {
	return fmt.Sprintf("%d", i)
}

func resourceUserCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	u := userBuildObject(d)
	items := zabbix.Users{u}

	err := api.UsersCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created user: %s", items[0].UserID)
	d.SetId(items[0].UserID)

	return resourceUserRead(d, m)
}

func resourceUserRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of user with id %s", d.Id())

	user, err := api.UserGetByID(d.Id())
	if err != nil {
		return err
	}
	if user == nil {
		d.SetId("")
		return nil
	}

	d.Set("username", user.Username)
	d.Set("name", user.Name)
	d.Set("surname", user.Surname)
	d.Set("roleid", user.RoleID)
	d.Set("lang", user.Lang)
	d.Set("timezone", user.Timezone)
	d.Set("theme", user.Theme)
	d.Set("autologin", user.Autologin == "1")
	d.Set("autologout", user.Autologout)
	d.Set("refresh", user.Refresh)
	d.Set("rows_per_page", user.RowsPerPage)
	d.Set("url", user.URL)

	// Do NOT set passwd - it is never returned by the API

	// usrgrps
	grpIDs := make([]string, len(user.UserGroups))
	for i, g := range user.UserGroups {
		grpIDs[i] = g.UserGroupID
	}
	d.Set("usrgrps", grpIDs)

	// medias
	mediaList := make([]map[string]interface{}, len(user.Medias))
	for i, media := range user.Medias {
		mediaList[i] = map[string]interface{}{
			"mediatypeid": media.MediaTypeID,
			"sendto":      media.SendTo,
			"active":      media.Active == "0", // 0 = enabled
			"severity":    userStrToInt(media.Severity),
			"period":      media.Period,
		}
	}
	d.Set("medias", mediaList)

	return nil
}

func userStrToInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func resourceUserUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	u := userBuildObject(d)
	u.UserID = d.Id()

	// Only send passwd on update if it was actually changed
	if !d.HasChange("passwd") {
		u.Passwd = ""
	}

	err := api.UsersUpdate(zabbix.Users{u})
	if err != nil {
		return err
	}

	return resourceUserRead(d, m)
}

func resourceUserDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.UsersDeleteByIds([]string{d.Id()})
}
