package provider

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

// resourceToken terraform resource handler for API tokens
func resourceToken() *schema.Resource {
	return &schema.Resource{
		Create: resourceTokenCreate,
		Read:   resourceTokenRead,
		Update: resourceTokenUpdate,
		Delete: resourceTokenDelete,
		// Import is deliberately refused. Zabbix only ever discloses a token
		// secret at generation time, so an imported token would land in state
		// with an empty "token" attribute. Because that attribute is computed,
		// Terraform would report no drift, leaving a resource that looks fully
		// managed but whose secret is unavailable. Refusing the import makes
		// that failure loud instead of silent.
		Importer: &schema.ResourceImporter{
			State: resourceTokenImportRefused,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Token name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"userid": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "ID of the user the token authenticates as, defaults to the calling user",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Token description",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the token may be used to authenticate",
			},
			"expires_at": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "0",
				Description: "Unix timestamp at which the token expires, 0 for never",
			},
			"token": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
				Description: "The generated token secret. Zabbix only returns this once, at " +
					"generation time, so it is stored in state and cannot be recovered on import.",
			},
			"lastaccess": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unix timestamp of the last time the token was used, 0 if never",
			},
			"created_at": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unix timestamp of token creation",
			},
		},
	}
}

// resourceTokenImportRefused rejects `terraform import` for zabbix_token.
//
// Importing would succeed mechanically but produce state with no secret, and
// the empty value would not show up as drift. Callers are pointed at the two
// safe alternatives instead.
func resourceTokenImportRefused(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	return nil, fmt.Errorf(
		"zabbix_token does not support import: Zabbix discloses a token secret only once, "+
			"when it is generated, so an imported token (id %s) would be managed without its "+
			"secret and Terraform could not detect that it is missing.\n"+
			"Use one of these instead:\n"+
			"  - data.zabbix_token, to reference an existing token's metadata (id, owner, "+
			"expiry) without its secret\n"+
			"  - a new zabbix_token resource, if you need a usable secret; note that "+
			"generating a secret for an existing token invalidates the previous one",
		d.Id())
}

// dataToken terraform data source for looking up an existing API token.
//
// This exposes metadata only. The secret is never returned by token.get, and is
// deliberately not offered here.
func dataToken() *schema.Resource {
	return &schema.Resource{
		Read: dataTokenRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Token name",
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"userid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: "ID of the user owning the token. Token names are only unique " +
					"per user, so set this when the same name may exist for several users.",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Token description",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the token may be used to authenticate",
			},
			"expires_at": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unix timestamp at which the token expires, 0 for never",
			},
			"lastaccess": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unix timestamp of the last time the token was used, 0 if never",
			},
			"created_at": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unix timestamp of token creation",
			},
		},
	}
}

// dataTokenRead looks up a token by name, optionally narrowed to one user.
func dataTokenRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	filter := map[string]interface{}{"name": d.Get("name").(string)}
	if v := d.Get("userid").(string); v != "" {
		filter["userid"] = v
	}

	tokens, err := api.TokensGet(zabbix.Params{"filter": filter})
	if err != nil {
		return err
	}

	if len(tokens) < 1 {
		return fmt.Errorf("no zabbix token found matching %v", filter)
	}
	if len(tokens) > 1 {
		return fmt.Errorf("%d zabbix tokens named %q found; token names are only unique per "+
			"user, set userid to disambiguate", len(tokens), d.Get("name").(string))
	}
	t := tokens[0]

	log.Debug("Got token: %s (%s)", t.Name, t.TokenID)

	d.SetId(t.TokenID)
	d.Set("name", t.Name)
	d.Set("userid", t.UserID)
	d.Set("description", t.Description)
	d.Set("enabled", t.Status == zabbix.TokenEnabled)
	d.Set("expires_at", t.ExpiresAt)
	d.Set("lastaccess", t.LastAccess)
	d.Set("created_at", t.CreatedAt)

	return nil
}

func buildTokenObject(d *schema.ResourceData) zabbix.Token {
	t := zabbix.Token{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ExpiresAt:   d.Get("expires_at").(string),
		Status:      zabbix.TokenDisabled,
	}
	if d.Get("enabled").(bool) {
		t.Status = zabbix.TokenEnabled
	}
	if v := d.Get("userid").(string); v != "" {
		t.UserID = v
	}
	return t
}

func resourceTokenCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	created, err := api.TokensCreate(zabbix.Tokens{buildTokenObject(d)})
	if err != nil {
		return err
	}
	if len(created) < 1 {
		return errors.New("token.create returned no token id")
	}

	id := created[0].TokenID
	d.SetId(id)
	log.Trace("created token: %s", id)

	// The secret only exists if we explicitly generate it, and Zabbix returns
	// it exactly once.
	secrets, err := api.TokensGenerate([]string{id})
	if err != nil {
		return err
	}
	if len(secrets) < 1 {
		return errors.New("token.generate returned no secret")
	}
	d.Set("token", secrets[0].Token)

	return resourceTokenRead(d, m)
}

func resourceTokenRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of token with id %s", d.Id())

	tokens, err := api.TokensGet(zabbix.Params{"tokenids": d.Id()})
	if err != nil {
		return err
	}
	if len(tokens) < 1 {
		d.SetId("")
		return nil
	}
	if len(tokens) > 1 {
		return errors.New("multiple tokens found")
	}
	t := tokens[0]

	d.SetId(t.TokenID)
	d.Set("name", t.Name)
	d.Set("description", t.Description)
	d.Set("userid", t.UserID)
	d.Set("enabled", t.Status == zabbix.TokenEnabled)
	d.Set("expires_at", t.ExpiresAt)
	d.Set("lastaccess", t.LastAccess)
	d.Set("created_at", t.CreatedAt)

	return nil
}

func resourceTokenUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	t := buildTokenObject(d)
	t.TokenID = d.Id()
	// userid is fixed at creation time; token.update rejects it
	t.UserID = ""

	if err := api.TokensUpdate(zabbix.Tokens{t}); err != nil {
		return err
	}

	return resourceTokenRead(d, m)
}

func resourceTokenDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.TokensDeleteByIds([]string{d.Id()})
}
