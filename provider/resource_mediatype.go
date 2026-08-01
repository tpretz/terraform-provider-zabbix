package provider

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var MEDIATYPE_TYPE_LOOKUP = map[string]string{
	"email":   "0",
	"script":  "1",
	"sms":     "2",
	"webhook": "4",
}
var MEDIATYPE_TYPE_LOOKUP_REV = map[string]string{
	"0": "email",
	"1": "script",
	"2": "sms",
	"4": "webhook",
}
var MEDIATYPE_TYPE_ARR = []string{"email", "script", "sms", "webhook"}

var MEDIATYPE_STATUS_LOOKUP = map[string]string{
	"enabled":  "0",
	"disabled": "1",
}
var MEDIATYPE_STATUS_LOOKUP_REV = map[string]string{
	"0": "enabled",
	"1": "disabled",
}
var MEDIATYPE_STATUS_ARR = []string{"enabled", "disabled"}

var MEDIATYPE_SMTP_SECURITY_LOOKUP = map[string]string{
	"none":     "0",
	"starttls": "1",
	"ssl_tls":  "2",
}
var MEDIATYPE_SMTP_SECURITY_LOOKUP_REV = map[string]string{
	"0": "none",
	"1": "starttls",
	"2": "ssl_tls",
}
var MEDIATYPE_SMTP_SECURITY_ARR = []string{"none", "starttls", "ssl_tls"}

var MEDIATYPE_SMTP_AUTH_LOOKUP = map[string]string{
	"none":     "0",
	"password": "1",
}
var MEDIATYPE_SMTP_AUTH_LOOKUP_REV = map[string]string{
	"0": "none",
	"1": "password",
}
var MEDIATYPE_SMTP_AUTH_ARR = []string{"none", "password"}

func resourceMediatype() *schema.Resource {
	return &schema.Resource{
		Create: resourceMediatypeCreate,
		Read:   resourceMediatypeRead,
		Update: resourceMediatypeUpdate,
		Delete: resourceMediatypeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Media type name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Media type: email, script, sms, webhook",
				ValidateFunc: validation.StringInSlice(MEDIATYPE_TYPE_ARR, false),
			},
			"status": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "enabled",
				Description:  "Status: enabled, disabled",
				ValidateFunc: validation.StringInSlice(MEDIATYPE_STATUS_ARR, false),
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the media type",
			},
			"max_sessions": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of alerts that can be processed in parallel (1 for email, 0-100 for others)",
			},
			"max_attempts": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of attempts to send an alert (1-100)",
			},
			"attempt_interval": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Interval between retry attempts (e.g. 10s)",
			},
			// Email fields
			"smtp_server": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SMTP server (email type)",
			},
			"smtp_port": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "SMTP port (email type)",
			},
			"smtp_helo": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SMTP HELO (email type)",
			},
			"smtp_email": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SMTP email sender address (email type)",
			},
			"smtp_security": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "none",
				Description:  "SMTP security: none, starttls, ssl_tls (email type)",
				ValidateFunc: validation.StringInSlice(MEDIATYPE_SMTP_SECURITY_ARR, false),
			},
			"smtp_authentication": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "none",
				Description:  "SMTP authentication: none, password (email type)",
				ValidateFunc: validation.StringInSlice(MEDIATYPE_SMTP_AUTH_ARR, false),
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Username for SMTP authentication (email type)",
			},
			"passwd": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "Password for SMTP authentication (email type)",
			},
			// Script fields
			"exec_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Script exec path (script type)",
			},
			"exec_params": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Script exec params (script type)",
			},
			// SMS fields
			"gsm_modem": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "GSM modem serial device path (sms type)",
			},
			// Webhook fields
			"script": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "Webhook JavaScript body (webhook type)",
			},
			"timeout": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Webhook timeout (webhook type)",
			},
			"process_tags": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to process event tags (webhook type)",
			},
			"show_event_menu": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Show entry in event menu (webhook type)",
			},
			"event_menu_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Event menu URL (webhook type)",
			},
			"event_menu_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Event menu entry name (webhook type)",
			},
			"parameters": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Webhook parameters (name/value pairs)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
			"message_templates": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Message templates",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"eventsource": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Event source: 0=triggers, 1=discovery, 2=autoregistration, 3=internal, 4=service",
						},
						"recovery": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Operation mode: 0=operations, 1=recovery, 2=update",
						},
						"subject": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"message": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
				},
			},
		},
	}
}

func mediatypeBuildObject(d *schema.ResourceData) zabbix.MediaType {
	mt := zabbix.MediaType{
		Name:               d.Get("name").(string),
		Type:               MEDIATYPE_TYPE_LOOKUP[d.Get("type").(string)],
		Status:             MEDIATYPE_STATUS_LOOKUP[d.Get("status").(string)],
		Description:        d.Get("description").(string),
		SmtpServer:         d.Get("smtp_server").(string),
		SmtpHelo:           d.Get("smtp_helo").(string),
		SmtpEmail:          d.Get("smtp_email").(string),
		SmtpSecurity:       MEDIATYPE_SMTP_SECURITY_LOOKUP[d.Get("smtp_security").(string)],
		SmtpAuthentication: MEDIATYPE_SMTP_AUTH_LOOKUP[d.Get("smtp_authentication").(string)],
		Username:           d.Get("username").(string),
		Passwd:             d.Get("passwd").(string),
		ExecPath:           d.Get("exec_path").(string),
		ExecParams:         d.Get("exec_params").(string),
		GsmModem:           d.Get("gsm_modem").(string),
		Script:             d.Get("script").(string),
		EventMenuURL:       d.Get("event_menu_url").(string),
		EventMenuName:      d.Get("event_menu_name").(string),
	}

	if v, ok := d.GetOk("max_sessions"); ok {
		mt.MaxSessions = v.(string)
	}
	if v, ok := d.GetOk("max_attempts"); ok {
		mt.MaxAttempts = v.(string)
	}
	if v, ok := d.GetOk("attempt_interval"); ok {
		mt.AttemptInterval = v.(string)
	}
	if v, ok := d.GetOk("smtp_port"); ok {
		mt.SmtpPort = v.(string)
	}
	if v, ok := d.GetOk("timeout"); ok {
		mt.Timeout = v.(string)
	}

	if d.Get("process_tags").(bool) {
		mt.ProcessTags = "1"
	} else {
		mt.ProcessTags = "0"
	}
	if d.Get("show_event_menu").(bool) {
		mt.ShowEventMenu = "1"
	} else {
		mt.ShowEventMenu = "0"
	}

	// Parameters
	if v, ok := d.GetOk("parameters"); ok {
		paramsList := v.([]interface{})
		params := make([]zabbix.MediaTypeParameter, len(paramsList))
		for i, p := range paramsList {
			pm := p.(map[string]interface{})
			params[i] = zabbix.MediaTypeParameter{
				Name:  pm["name"].(string),
				Value: pm["value"].(string),
			}
		}
		mt.Parameters = params
	}

	// Message templates
	if v, ok := d.GetOk("message_templates"); ok {
		mtList := v.([]interface{})
		templates := make([]zabbix.MediaTypeMessageTemplate, len(mtList))
		for i, t := range mtList {
			tm := t.(map[string]interface{})
			templates[i] = zabbix.MediaTypeMessageTemplate{
				EventSource: tm["eventsource"].(string),
				Recovery:    tm["recovery"].(string),
				Subject:     tm["subject"].(string),
				Message:     tm["message"].(string),
			}
		}
		mt.MessageTemplates = templates
	}

	return mt
}

func resourceMediatypeCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := mediatypeBuildObject(d)
	items := zabbix.MediaTypes{item}

	err := api.MediaTypesCreate(items)
	if err != nil {
		return err
	}

	log.Trace("created mediatype: %+v", items[0])
	d.SetId(items[0].MediaTypeID)

	return resourceMediatypeRead(d, m)
}

func resourceMediatypeRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of mediatype with id %s", d.Id())

	mts, err := api.MediaTypesGet(zabbix.Params{
		"mediatypeids":           d.Id(),
		"selectMessageTemplates": "extend",
		"selectParameters":       "extend",
	})
	if err != nil {
		return err
	}

	if len(mts) < 1 {
		d.SetId("")
		return nil
	}
	if len(mts) > 1 {
		return errors.New("multiple media types found")
	}
	mt := mts[0]

	d.SetId(mt.MediaTypeID)
	d.Set("name", mt.Name)
	d.Set("type", MEDIATYPE_TYPE_LOOKUP_REV[mt.Type])
	d.Set("status", MEDIATYPE_STATUS_LOOKUP_REV[mt.Status])
	d.Set("description", mt.Description)
	d.Set("max_sessions", mt.MaxSessions)
	d.Set("max_attempts", mt.MaxAttempts)
	d.Set("attempt_interval", mt.AttemptInterval)
	d.Set("smtp_server", mt.SmtpServer)
	d.Set("smtp_port", mt.SmtpPort)
	d.Set("smtp_helo", mt.SmtpHelo)
	d.Set("smtp_email", mt.SmtpEmail)
	d.Set("smtp_security", MEDIATYPE_SMTP_SECURITY_LOOKUP_REV[mt.SmtpSecurity])
	d.Set("smtp_authentication", MEDIATYPE_SMTP_AUTH_LOOKUP_REV[mt.SmtpAuthentication])
	d.Set("username", mt.Username)
	// passwd is never returned by the API, keep the state value
	d.Set("exec_path", mt.ExecPath)
	d.Set("exec_params", mt.ExecParams)
	d.Set("gsm_modem", mt.GsmModem)
	// script is sensitive but still returned by the API
	d.Set("script", mt.Script)
	d.Set("timeout", mt.Timeout)
	d.Set("process_tags", mt.ProcessTags == "1")
	d.Set("show_event_menu", mt.ShowEventMenu == "1")
	d.Set("event_menu_url", mt.EventMenuURL)
	d.Set("event_menu_name", mt.EventMenuName)

	// Parameters
	params := make([]map[string]interface{}, len(mt.Parameters))
	for i, p := range mt.Parameters {
		params[i] = map[string]interface{}{
			"name":  p.Name,
			"value": p.Value,
		}
	}
	d.Set("parameters", params)

	// Message templates
	templates := make([]map[string]interface{}, len(mt.MessageTemplates))
	for i, t := range mt.MessageTemplates {
		templates[i] = map[string]interface{}{
			"eventsource": t.EventSource,
			"recovery":    t.Recovery,
			"subject":     t.Subject,
			"message":     t.Message,
		}
	}
	d.Set("message_templates", templates)

	return nil
}

func resourceMediatypeUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	item := mediatypeBuildObject(d)
	item.MediaTypeID = d.Id()

	err := api.MediaTypesUpdate(zabbix.MediaTypes{item})
	if err != nil {
		return err
	}

	return resourceMediatypeRead(d, m)
}

func resourceMediatypeDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.MediaTypesDeleteByIds([]string{d.Id()})
}
