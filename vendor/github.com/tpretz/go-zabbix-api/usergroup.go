package zabbix

// UserGroup represents a Zabbix user group object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usergroup/object
type UserGroup struct {
	UserGroupID string `json:"usrgrpid,omitempty"`
	Name        string `json:"name"`
	GuiAccess   string `json:"gui_access,omitempty"`
	UsersStatus string `json:"users_status,omitempty"`
	DebugMode   string `json:"debug_mode,omitempty"`

	// Permissions
	HostGroupRights     []Permission `json:"hostgroup_rights,omitempty"`
	TemplateGroupRights []Permission `json:"templategroup_rights,omitempty"`
	TagFilters          []TagFilter  `json:"tag_filters,omitempty"`
}

// UserGroups is an array of UserGroup
type UserGroups []UserGroup

// Permission represents a host/template group permission entry.
type Permission struct {
	ID         string `json:"id"`
	Permission string `json:"permission"`
}

// TagFilter represents a tag-based permission entry.
type TagFilter struct {
	GroupID string `json:"groupid"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
}

// UserGroupsGet Wrapper for usergroup.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usergroup/get
func (api *API) UserGroupsGet(params Params) (res UserGroups, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("usergroup.get", params, &res)
	return
}

// UserGroupGetByID Gets a user group by ID only if there is exactly 1 matching group.
func (api *API) UserGroupGetByID(id string) (res *UserGroup, err error) {
	groups, err := api.UserGroupsGet(Params{
		"usrgrpids":                 id,
		"selectHostGroupRights":     "extend",
		"selectTemplateGroupRights": "extend",
		"selectTagFilters":          "extend",
	})
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

// UserGroupsCreate Wrapper for usergroup.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usergroup/create
func (api *API) UserGroupsCreate(groups UserGroups) (err error) {
	response, err := api.CallWithError("usergroup.create", groups)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["usrgrpids"].([]interface{})
	for i, id := range ids {
		groups[i].UserGroupID = id.(string)
	}
	return
}

// UserGroupsUpdate Wrapper for usergroup.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usergroup/update
func (api *API) UserGroupsUpdate(groups UserGroups) (err error) {
	_, err = api.CallWithError("usergroup.update", groups)
	return
}

// UserGroupsDeleteByIds Wrapper for usergroup.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usergroup/delete
func (api *API) UserGroupsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("usergroup.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	deleted := result["usrgrpids"].([]interface{})
	if len(ids) != len(deleted) {
		err = &ExpectedMore{len(ids), len(deleted)}
	}
	return
}
