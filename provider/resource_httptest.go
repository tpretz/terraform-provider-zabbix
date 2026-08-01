package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
	"strconv"
	"strings"
)

var HTTPTEST_STATUS_LOOKUP = map[string]string{
	"enabled":  "0",
	"disabled": "1",
}
var HTTPTEST_STATUS_LOOKUP_REV = map[string]string{}
var HTTPTEST_STATUS_ARR = []string{}

var HTTPTEST_AUTH_LOOKUP = map[string]string{
	"none":     "0",
	"basic":    "1",
	"ntlm":     "2",
	"kerberos": "3",
	"digest":   "4",
}
var HTTPTEST_AUTH_LOOKUP_REV = map[string]string{}
var HTTPTEST_AUTH_ARR = []string{}

var _ = func() bool {
	for k, v := range HTTPTEST_STATUS_LOOKUP {
		HTTPTEST_STATUS_LOOKUP_REV[v] = k
		HTTPTEST_STATUS_ARR = append(HTTPTEST_STATUS_ARR, k)
	}
	for k, v := range HTTPTEST_AUTH_LOOKUP {
		HTTPTEST_AUTH_LOOKUP_REV[v] = k
		HTTPTEST_AUTH_ARR = append(HTTPTEST_AUTH_ARR, k)
	}
	return false
}()

// httptestBool converts a terraform bool into the "0"/"1" string the API uses.
func httptestBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func resourceHTTPTest() *schema.Resource {
	return &schema.Resource{
		Create: resourceHTTPTestCreate,
		Read:   resourceHTTPTestRead,
		Update: resourceHTTPTestUpdate,
		Delete: resourceHTTPTestDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Web scenario name",
			},
			"hostid": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the host the web scenario belongs to",
			},
			"delay": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "60",
				Description: "Execution interval (default 60s)",
			},
			"retries": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1",
				Description: "Number of retries (1-10)",
			},
			"agent": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Zabbix",
				Description: "HTTP user agent string",
			},
			"http_proxy": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "HTTP proxy string",
			},
			"authentication": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "none",
				Description:  "HTTP authentication type, one of: " + strings.Join(HTTPTEST_AUTH_ARR, ", "),
				ValidateFunc: validation.StringInSlice(HTTPTEST_AUTH_ARR, false),
			},
			"http_user": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "HTTP authentication user name",
			},
			"http_password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Sensitive:   true,
				Description: "HTTP authentication password",
			},
			"verify_peer": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Verify the SSL certificate of the peer",
			},
			"verify_host": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Verify the host name in the SSL certificate",
			},
			"status": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "enabled",
				Description:  "Status, one of: " + strings.Join(HTTPTEST_STATUS_ARR, ", "),
				ValidateFunc: validation.StringInSlice(HTTPTEST_STATUS_ARR, false),
			},
			"variables": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Scenario-level variables",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"headers": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Scenario-level HTTP headers",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"steps": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Web scenario steps (at least one required)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"url": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"no": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "Step order number",
						},
						"timeout": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "15",
						},
						"posts": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"required": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"status_codes": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"follow_redirects": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "1",
						},
						"retrieve_mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "0",
						},
						"headers": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"variables": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"query_fields": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func httptestBuildHeaders(raw []interface{}) zabbix.HTTPTestHeaders {
	headers := make(zabbix.HTTPTestHeaders, len(raw))
	for i, h := range raw {
		m := h.(map[string]interface{})
		headers[i] = zabbix.HTTPTestHeader{
			Name:  m["name"].(string),
			Value: m["value"].(string),
		}
	}
	return headers
}

func httptestBuildObject(d *schema.ResourceData) zabbix.HTTPTest {
	ht := zabbix.HTTPTest{
		Name:           d.Get("name").(string),
		HostID:         d.Get("hostid").(string),
		Delay:          d.Get("delay").(string),
		Retries:        d.Get("retries").(string),
		Agent:          d.Get("agent").(string),
		HTTPProxy:      d.Get("http_proxy").(string),
		Authentication: HTTPTEST_AUTH_LOOKUP[d.Get("authentication").(string)],
		HTTPUser:       d.Get("http_user").(string),
		HTTPPassword:   d.Get("http_password").(string),
		VerifyPeer:     httptestBool(d.Get("verify_peer").(bool)),
		VerifyHost:     httptestBool(d.Get("verify_host").(bool)),
		Status:         HTTPTEST_STATUS_LOOKUP[d.Get("status").(string)],
	}

	// variables
	if v, ok := d.GetOk("variables"); ok {
		ht.Variables = httptestBuildHeaders(v.([]interface{}))
	}

	// headers
	if v, ok := d.GetOk("headers"); ok {
		ht.Headers = httptestBuildHeaders(v.([]interface{}))
	}

	// steps
	stepsRaw := d.Get("steps").([]interface{})
	ht.Steps = make(zabbix.HTTPTestSteps, len(stepsRaw))
	for i, raw := range stepsRaw {
		s := raw.(map[string]interface{})
		step := zabbix.HTTPTestStep{
			Name:            s["name"].(string),
			URL:             s["url"].(string),
			No:              s["no"].(string),
			Timeout:         s["timeout"].(string),
			Posts:           s["posts"].(string),
			Required:        s["required"].(string),
			StatusCodes:     s["status_codes"].(string),
			FollowRedirects: s["follow_redirects"].(string),
			RetrieveMode:    s["retrieve_mode"].(string),
		}
		if h, ok := s["headers"]; ok {
			step.Headers = httptestBuildHeaders(h.([]interface{}))
		}
		if v, ok := s["variables"]; ok {
			step.Variables = httptestBuildHeaders(v.([]interface{}))
		}
		if q, ok := s["query_fields"]; ok {
			step.QueryFields = httptestBuildHeaders(q.([]interface{}))
		}
		ht.Steps[i] = step
	}

	return ht
}

func httptestFlattenHeaders(headers zabbix.HTTPTestHeaders) []map[string]interface{} {
	result := make([]map[string]interface{}, len(headers))
	for i, h := range headers {
		result[i] = map[string]interface{}{
			"name":  h.Name,
			"value": h.Value,
		}
	}
	return result
}

func resourceHTTPTestCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	ht := httptestBuildObject(d)
	tests := zabbix.HTTPTests{ht}

	err := api.HTTPTestsCreate(tests)
	if err != nil {
		return err
	}

	d.SetId(tests[0].HTTPTestID)
	log.Trace("created httptest: %s", tests[0].HTTPTestID)

	return resourceHTTPTestRead(d, m)
}

func resourceHTTPTestRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of httptest with id %s", d.Id())

	ht, err := api.HTTPTestGetByID(d.Id())
	if err != nil {
		return err
	}
	if ht == nil {
		d.SetId("")
		return nil
	}

	d.SetId(ht.HTTPTestID)
	d.Set("name", ht.Name)
	d.Set("hostid", ht.HostID)
	d.Set("delay", ht.Delay)
	d.Set("retries", ht.Retries)
	d.Set("agent", ht.Agent)
	d.Set("http_proxy", ht.HTTPProxy)
	d.Set("authentication", HTTPTEST_AUTH_LOOKUP_REV[ht.Authentication])
	d.Set("http_user", ht.HTTPUser)
	d.Set("http_password", ht.HTTPPassword)
	d.Set("verify_peer", ht.VerifyPeer == "1")
	d.Set("verify_host", ht.VerifyHost == "1")
	d.Set("status", HTTPTEST_STATUS_LOOKUP_REV[ht.Status])

	// variables
	d.Set("variables", httptestFlattenHeaders(ht.Variables))

	// headers
	d.Set("headers", httptestFlattenHeaders(ht.Headers))

	// steps
	steps := make([]map[string]interface{}, len(ht.Steps))
	for i, s := range ht.Steps {
		step := map[string]interface{}{
			"name":             s.Name,
			"url":              s.URL,
			"no":               s.No,
			"timeout":          s.Timeout,
			"posts":            s.Posts,
			"required":         s.Required,
			"status_codes":     s.StatusCodes,
			"follow_redirects": s.FollowRedirects,
			"retrieve_mode":    s.RetrieveMode,
			"headers":          httptestFlattenHeaders(s.Headers),
			"variables":        httptestFlattenHeaders(s.Variables),
			"query_fields":     httptestFlattenHeaders(s.QueryFields),
		}
		steps[i] = step
	}
	d.Set("steps", steps)

	return nil
}

func resourceHTTPTestUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	ht := httptestBuildObject(d)
	ht.HTTPTestID = d.Id()
	// hostid is rejected on update
	ht.HostID = ""

	// We need to set httpstepid on existing steps for update
	// Read current state to get step IDs
	existing, err := api.HTTPTestGetByID(d.Id())
	if err != nil {
		return err
	}

	// Map step no -> httpstepid from existing
	stepIDMap := make(map[string]string)
	if existing != nil {
		for _, s := range existing.Steps {
			stepIDMap[s.No] = s.HTTPStepID
		}
	}

	// Assign httpstepid to matching steps
	for i := range ht.Steps {
		if id, ok := stepIDMap[ht.Steps[i].No]; ok {
			ht.Steps[i].HTTPStepID = id
		}
	}

	// Explicitly clear arrays if empty
	if _, ok := d.GetOk("variables"); !ok {
		ht.Variables = zabbix.HTTPTestHeaders{}
	}
	if _, ok := d.GetOk("headers"); !ok {
		ht.Headers = zabbix.HTTPTestHeaders{}
	}

	// Clear read-only fields from steps
	for i := range ht.Steps {
		ht.Steps[i].HTTPTestID = ""
		ht.Steps[i].PostType = ""
	}

	if err := api.HTTPTestsUpdate(zabbix.HTTPTests{ht}); err != nil {
		return err
	}

	return resourceHTTPTestRead(d, m)
}

func resourceHTTPTestDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.HTTPTestsDeleteByIds([]string{d.Id()})
}

// httptestStepNo generates step number string from int
func httptestStepNo(n int) string {
	return strconv.Itoa(n)
}
