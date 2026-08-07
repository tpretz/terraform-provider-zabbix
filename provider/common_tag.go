package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// tag list schema
// tagHash is the set hash for tagSetSchema. It has to cover both key and value,
// because helper/schema's diffSet compares hash codes only and never inspects
// elements whose codes match -- an attribute left out of the hash can never be
// seen to change, and the edit is silently discarded. See CLAUDE.md.
func tagHash(i interface{}) int {
	m := i.(map[string]interface{})
	return schema.HashString(m["key"].(string) + "V" + m["value"].(string))
}

var tagSetSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Set:      tagHash,
	Description: "Tags applied to this object (unordered). Tags are how Zabbix 5.4 and later " +
		"group and filter objects; they replaced applications. Zabbix replaces the whole tag " +
		"collection on update, so omitting a tag removes it.",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Tag name. Zabbix allows several tags to share a name with different values.",
			},
			"value": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Tag value. Optional: a tag with no value is a valid tag.",
			},
		},
	},
}

// tagGenerate build tag structs from terraform inputs
func tagGenerate(d *schema.ResourceData) (tags zabbix.Tags) {
	set := d.Get("tag").(*schema.Set).List()
	tags = make(zabbix.Tags, len(set))

	for i := 0; i < len(set); i++ {
		current := set[i].(map[string]interface{})
		tags[i] = zabbix.Tag{
			Tag:   current["key"].(string),
			Value: current["value"].(string),
		}
	}

	return
}

// flattenTags convert response to terraform input
func flattenTags(list zabbix.Tags) *schema.Set {
	set := schema.NewSet(tagHash, []interface{}{})
	for i := 0; i < len(list); i++ {
		set.Add(map[string]interface{}{
			"key":   list[i].Tag,
			"value": list[i].Value,
		})
	}
	return set
}
