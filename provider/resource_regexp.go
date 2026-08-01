package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/tpretz/go-zabbix-api"
)

var REGEXP_EXPRESSION_TYPE_LOOKUP = map[string]string{
	"char_included":     "0",
	"char_not_included": "1",
	"regexp":            "2",
	"not_regexp":        "3",
	"any_included":      "4",
}

var REGEXP_EXPRESSION_TYPE_LOOKUP_REV = map[string]string{
	"0": "char_included",
	"1": "char_not_included",
	"2": "regexp",
	"3": "not_regexp",
	"4": "any_included",
}

var REGEXP_EXPRESSION_TYPE_ARR = []string{
	"char_included",
	"char_not_included",
	"regexp",
	"not_regexp",
	"any_included",
}

func resourceRegexp() *schema.Resource {
	return &schema.Resource{
		Create: resourceRegexpCreate,
		Read:   resourceRegexpRead,
		Update: resourceRegexpUpdate,
		Delete: resourceRegexpDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Name of the regular expression",
			},
			"test_string": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Test string for the regular expression",
			},
			"expressions": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: "Regular expression sub-expressions",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"expression": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "The expression string",
						},
						"expression_type": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(REGEXP_EXPRESSION_TYPE_ARR, false),
							Description:  "Expression type",
						},
						"exp_delimiter": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "",
							Description: "Expression delimiter (must be empty in Zabbix 7.0+)",
						},
						"case_sensitive": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Whether the expression is case sensitive",
						},
					},
				},
			},
		},
	}
}

func regexpBuildExpressions(d *schema.ResourceData) zabbix.RegexpExpressions {
	raw := d.Get("expressions").([]interface{})
	expressions := make(zabbix.RegexpExpressions, len(raw))
	for i, v := range raw {
		m := v.(map[string]interface{})
		caseSensitive := "0"
		if m["case_sensitive"].(bool) {
			caseSensitive = "1"
		}
		expressions[i] = zabbix.RegexpExpression{
			Expression:     m["expression"].(string),
			ExpressionType: zabbix.RegexpExpressionType(REGEXP_EXPRESSION_TYPE_LOOKUP[m["expression_type"].(string)]),
			ExpDelimiter:   m["exp_delimiter"].(string),
			CaseSensitive:  caseSensitive,
		}
	}
	return expressions
}

func regexpFlattenExpressions(expressions zabbix.RegexpExpressions) []map[string]interface{} {
	result := make([]map[string]interface{}, len(expressions))
	for i, e := range expressions {
		result[i] = map[string]interface{}{
			"expression":      e.Expression,
			"expression_type": REGEXP_EXPRESSION_TYPE_LOOKUP_REV[string(e.ExpressionType)],
			"exp_delimiter":   e.ExpDelimiter,
			"case_sensitive":  e.CaseSensitive == "1",
		}
	}
	return result
}

func resourceRegexpCreate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	r := zabbix.Regexp{
		Name:        d.Get("name").(string),
		TestString:  d.Get("test_string").(string),
		Expressions: regexpBuildExpressions(d),
	}

	regexps := zabbix.Regexps{r}
	err := api.RegexpsCreate(regexps)
	if err != nil {
		return err
	}

	log.Trace("created regexp: %+v", regexps[0])
	d.SetId(regexps[0].RegexpID)

	return resourceRegexpRead(d, m)
}

func resourceRegexpRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	log.Debug("Lookup of regexp with id %s", d.Id())

	r, err := api.RegexpGetByID(d.Id())
	if err != nil {
		return err
	}
	if r == nil {
		d.SetId("")
		return nil
	}

	d.SetId(r.RegexpID)
	d.Set("name", r.Name)
	d.Set("test_string", r.TestString)
	d.Set("expressions", regexpFlattenExpressions(r.Expressions))

	return nil
}

func resourceRegexpUpdate(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)

	r := zabbix.Regexp{
		RegexpID:    d.Id(),
		Name:        d.Get("name").(string),
		TestString:  d.Get("test_string").(string),
		Expressions: regexpBuildExpressions(d),
	}

	err := api.RegexpsUpdate(zabbix.Regexps{r})
	if err != nil {
		return err
	}

	return resourceRegexpRead(d, m)
}

func resourceRegexpDelete(d *schema.ResourceData, m interface{}) error {
	api := m.(*zabbix.API)
	return api.RegexpsDeleteByIds([]string{d.Id()})
}
