package zabbix

// HostPrototypeGroupPrototype is a dynamic group name pattern, e.g. "{#HV.NAME}".
type HostPrototypeGroupPrototype struct {
	GroupPrototypeID string `json:"group_prototypeid,omitempty"`
	Name             string `json:"name"`
}

// HostPrototypeGroupPrototypes is an array of HostPrototypeGroupPrototype.
type HostPrototypeGroupPrototypes []HostPrototypeGroupPrototype

// HostPrototype represents a Zabbix host prototype.
// https://www.zabbix.com/documentation/current/en/manual/api/reference/hostprototype/object
//
// Does not cover inventory_mode, interfaces, macros, monitored_by or proxy assignment.
type HostPrototype struct {
	HostID string `json:"hostid,omitempty"`

	// Create-only, like Item.RuleID; cleared before every update call.
	RuleID string `json:"ruleid,omitempty"`

	// Read-only (selectDiscoveryRule); hostprototype.get has no bare "ruleid"
	// property at all, so this is the only way to read the owning rule back.
	DiscoveryRule *LLDRule `json:"discoveryRule,omitempty"`

	Host string `json:"host"`
	Name string `json:"name"`

	Status StatusType `json:"status,string"`

	// 0 = discovered, 1 = not discovered / mark lost.
	Discover int `json:"discover,string"`

	// No omitempty: Zabbix replaces these wholesale on update. prepHostPrototypes nils->[].
	GroupLinks      HostGroupIDs                 `json:"groupLinks"`
	GroupPrototypes HostPrototypeGroupPrototypes `json:"groupPrototypes"`
	TemplateIDs     TemplateIDs                  `json:"templates"`
	Tags            Tags                         `json:"tags"`
}

// HostPrototypes is an array of HostPrototype.
type HostPrototypes []HostPrototype

func prepHostPrototypes(items HostPrototypes) {
	for i := range items {
		if items[i].GroupLinks == nil {
			items[i].GroupLinks = HostGroupIDs{}
		}
		if items[i].GroupPrototypes == nil {
			items[i].GroupPrototypes = HostPrototypeGroupPrototypes{}
		}
		if items[i].TemplateIDs == nil {
			items[i].TemplateIDs = TemplateIDs{}
		}
		if items[i].Tags == nil {
			items[i].Tags = Tags{}
		}
	}
}

func prepHostPrototypesUpdate(items HostPrototypes) {
	prepHostPrototypes(items)
	for i := range items {
		items[i].RuleID = ""
	}
}

// HostPrototypesGet wraps hostprototype.get.
func (api *API) HostPrototypesGet(params Params) (res HostPrototypes, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("hostprototype.get", params, &res)
	return
}

// HostPrototypeGetByID gets a host prototype by id, only if exactly one matches.
func (api *API) HostPrototypeGetByID(id string) (res *HostPrototype, err error) {
	items, err := api.HostPrototypesGet(Params{"hostids": id})
	if err != nil {
		return
	}
	if len(items) != 1 {
		e := ExpectedOneResult(len(items))
		err = &e
		return
	}
	res = &items[0]
	return
}

// HostPrototypesCreate wraps hostprototype.create.
func (api *API) HostPrototypesCreate(items HostPrototypes) (err error) {
	prepHostPrototypes(items)
	response, err := api.CallWithError("hostprototype.create", items)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	hostids := result["hostids"].([]interface{})
	for i, id := range hostids {
		items[i].HostID = id.(string)
	}
	return
}

// HostPrototypesUpdate wraps hostprototype.update.
func (api *API) HostPrototypesUpdate(items HostPrototypes) (err error) {
	prepHostPrototypesUpdate(items)
	_, err = api.CallWithError("hostprototype.update", items)
	return
}

// HostPrototypesDeleteByIds wraps hostprototype.delete.
func (api *API) HostPrototypesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("hostprototype.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	deleted, ok := result["hostids"].([]interface{})
	if !ok {
		return
	}
	if len(deleted) != len(ids) {
		err = &ExpectedMore{len(ids), len(deleted)}
	}
	return
}
