package zabbix

// TemplateGroup represent Zabbix template group object.
//
// Template groups were split out of host groups in Zabbix 6.2. On Zabbix 6.2
// and later, templates belong to template groups, which live in a separate
// ID space from host groups.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/templategroup/object
type TemplateGroup struct {
	GroupID string `json:"groupid,omitempty"`
	Name    string `json:"name"`
	UUID    string `json:"uuid,omitempty"`
}

// TemplateGroups is an array of TemplateGroup
type TemplateGroups []TemplateGroup

// TemplateGroupsGet Wrapper for templategroup.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/templategroup/get
func (api *API) TemplateGroupsGet(params Params) (res TemplateGroups, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("templategroup.get", params, &res)
	return
}

// TemplateGroupGetByID Gets template group by ID only if there is exactly 1 matching template group.
func (api *API) TemplateGroupGetByID(id string) (res *TemplateGroup, err error) {
	groups, err := api.TemplateGroupsGet(Params{"groupids": id})
	if err != nil {
		return
	}

	if len(groups) == 1 {
		res = &groups[0]
	} else {
		e := ExpectedOneResult(len(groups))
		err = &e
	}
	return
}

// TemplateGroupsCreate Wrapper for templategroup.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/templategroup/create
func (api *API) TemplateGroupsCreate(groups TemplateGroups) (err error) {
	response, err := api.CallWithError("templategroup.create", groups)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["groupids"].([]interface{})
	for i, id := range groupids {
		groups[i].GroupID = id.(string)
	}
	return
}

// TemplateGroupsUpdate Wrapper for templategroup.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/templategroup/update
func (api *API) TemplateGroupsUpdate(groups TemplateGroups) (err error) {
	_, err = api.CallWithError("templategroup.update", groups)
	return
}

// TemplateGroupsDeleteByIds Wrapper for templategroup.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/templategroup/delete
func (api *API) TemplateGroupsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("templategroup.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["groupids"].([]interface{})
	if len(ids) != len(groupids) {
		err = &ExpectedMore{len(ids), len(groupids)}
	}
	return
}

// TemplateGroupsDelete Wrapper for templategroup.delete
// Cleans GroupID in all groups elements if call succeed.
func (api *API) TemplateGroupsDelete(groups TemplateGroups) (err error) {
	ids := make([]string, len(groups))
	for i, group := range groups {
		ids[i] = group.GroupID
	}

	err = api.TemplateGroupsDeleteByIds(ids)
	if err == nil {
		for i := range groups {
			groups[i].GroupID = ""
		}
	}
	return
}
