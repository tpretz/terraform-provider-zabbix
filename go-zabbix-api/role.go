package zabbix

// Role represents a Zabbix role object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/role/object
type Role struct {
	RoleID   string    `json:"roleid,omitempty"`
	Name     string    `json:"name"`
	Type     string    `json:"type,omitempty"`
	ReadOnly string    `json:"readonly,omitempty"`
	Rules    *RoleRule `json:"rules,omitempty"`
}

// Roles is an array of Role
type Roles []Role

// RoleRule represents rules assigned to a role.
type RoleRule struct {
	UI                  []RoleRuleUIElement `json:"ui,omitempty"`
	UIDefaultAccess     string              `json:"ui.default_access,omitempty"`
	Actions             []RoleRuleAction    `json:"actions,omitempty"`
	ActionsDefaultAcces string              `json:"actions.default_access,omitempty"`
	APIAccess           string              `json:"api.access,omitempty"`
	APIMode             string              `json:"api.mode,omitempty"`
	API                 []string            `json:"api"`
}

// RoleRuleUIElement represents a UI element toggle.
type RoleRuleUIElement struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RoleRuleAction represents an action toggle.
type RoleRuleAction struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RolesGet Wrapper for role.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/role/get
func (api *API) RolesGet(params Params) (res Roles, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("role.get", params, &res)
	return
}

// RoleGetByID Gets a role by ID only if there is exactly 1 matching role.
func (api *API) RoleGetByID(id string) (res *Role, err error) {
	roles, err := api.RolesGet(Params{
		"roleids":     id,
		"selectRules": "extend",
	})
	if err != nil {
		return
	}

	if len(roles) == 1 {
		res = &roles[0]
	} else {
		e := ExpectedOneResult(len(roles))
		err = &e
	}
	return
}

// RolesCreate Wrapper for role.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/role/create
func (api *API) RolesCreate(roles Roles) (err error) {
	response, err := api.CallWithError("role.create", roles)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["roleids"].([]interface{})
	for i, id := range ids {
		roles[i].RoleID = id.(string)
	}
	return
}

// RolesUpdate Wrapper for role.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/role/update
func (api *API) RolesUpdate(roles Roles) (err error) {
	_, err = api.CallWithError("role.update", roles)
	return
}

// RolesDeleteByIds Wrapper for role.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/role/delete
func (api *API) RolesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("role.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	deleted := result["roleids"].([]interface{})
	if len(ids) != len(deleted) {
		err = &ExpectedMore{len(ids), len(deleted)}
	}
	return
}
