package zabbix

// Script represents a Zabbix script object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/script/object
type Script struct {
	ScriptID string `json:"scriptid,omitempty"`
	Name     string `json:"name"`
	Command  string `json:"command,omitempty"`
	Type     string `json:"type"`
	// immutable after create; cleared on update, so omitempty is required or
	// Zabbix receives "" and reports "an integer is expected"
	Scope        string            `json:"scope,omitempty"`
	ExecuteOn    string            `json:"execute_on,omitempty"`
	Description  string            `json:"description,omitempty"`
	Confirmation string            `json:"confirmation,omitempty"`
	HostAccess   string            `json:"host_access,omitempty"`
	GroupID      string            `json:"groupid,omitempty"`
	UsrGrpID     string            `json:"usrgrpid,omitempty"`
	MenuPath     string            `json:"menu_path,omitempty"`
	Timeout      string            `json:"timeout,omitempty"`
	Parameters   []ScriptParameter `json:"parameters,omitempty"`
}

// ScriptParameter represents a webhook parameter for a script.
type ScriptParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Scripts is an array of Script
type Scripts []Script

// ScriptsGet Wrapper for script.get
func (api *API) ScriptsGet(params Params) (res Scripts, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("script.get", params, &res)
	return
}

// ScriptGetByID Gets a script by ID only if there is exactly 1 matching script.
func (api *API) ScriptGetByID(id string) (res *Script, err error) {
	scripts, err := api.ScriptsGet(Params{"scriptids": id})
	if err != nil {
		return
	}

	if len(scripts) == 1 {
		res = &scripts[0]
	} else {
		e := ExpectedOneResult(len(scripts))
		err = &e
	}
	return
}

// ScriptsCreate Wrapper for script.create
func (api *API) ScriptsCreate(scripts Scripts) (err error) {
	response, err := api.CallWithError("script.create", scripts)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["scriptids"].([]interface{})
	for i, id := range ids {
		scripts[i].ScriptID = id.(string)
	}
	return
}

// ScriptsUpdate Wrapper for script.update
func (api *API) ScriptsUpdate(scripts Scripts) (err error) {
	_, err = api.CallWithError("script.update", scripts)
	return
}

// ScriptsDeleteByIds Wrapper for script.delete
func (api *API) ScriptsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("script.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	scriptids := result["scriptids"].([]interface{})
	if len(ids) != len(scriptids) {
		err = &ExpectedMore{len(ids), len(scriptids)}
	}
	return
}
