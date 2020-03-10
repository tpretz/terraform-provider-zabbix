package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tpretz/go-zabbix-api"
)

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

func buildTriggerIds(s *schema.Set) zabbix.TriggerIDs {
	list := s.List()

	groups := make(zabbix.TriggerIDs, len(list))

	for i := 0; i < len(list); i++ {
		groups[i] = zabbix.TriggerID{
			TriggerID: list[i].(string),
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

// mergeSchemas, take a varadic list of schemas and merge, latter overwrites former
func mergeSchemas(schemas ...map[string]*schema.Schema) map[string]*schema.Schema {
	n := map[string]*schema.Schema{}

	for _, s := range schemas {
		for k, v := range s {
			n[k] = v
		}
	}

	return n
}
