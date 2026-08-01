package zabbix

// GlobalMacroType represents the type of a global macro
type GlobalMacroType string

const (
	GlobalMacroText   GlobalMacroType = "0"
	GlobalMacroSecret GlobalMacroType = "1"
	GlobalMacroVault  GlobalMacroType = "2"
)

// GlobalMacro represents a Zabbix global macro object
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usermacro/object#global_macro
type GlobalMacro struct {
	GlobalMacroID string          `json:"globalmacroid,omitempty"`
	Macro         string          `json:"macro"`
	Value         string          `json:"value,omitempty"`
	Type          GlobalMacroType `json:"type,omitempty"`
	Description   string          `json:"description,omitempty"`
}

// GlobalMacros is an array of GlobalMacro
type GlobalMacros []GlobalMacro

// GlobalMacrosGet Wrapper for usermacro.get with globalmacro: true
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usermacro/get
func (api *API) GlobalMacrosGet(params Params) (res GlobalMacros, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	params["globalmacro"] = true
	err = api.CallWithErrorParse("usermacro.get", params, &res)
	return
}

// GlobalMacroGetByID Gets a global macro by ID only if there is exactly 1 matching.
func (api *API) GlobalMacroGetByID(id string) (res *GlobalMacro, err error) {
	macros, err := api.GlobalMacrosGet(Params{"globalmacroids": []string{id}})
	if err != nil {
		return
	}

	if len(macros) == 1 {
		res = &macros[0]
	} else {
		e := ExpectedOneResult(len(macros))
		err = &e
	}
	return
}

// GlobalMacrosCreate Wrapper for usermacro.createglobal
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usermacro/createglobal
func (api *API) GlobalMacrosCreate(macros GlobalMacros) (err error) {
	response, err := api.CallWithError("usermacro.createglobal", macros)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["globalmacroids"].([]interface{})
	for i, id := range ids {
		macros[i].GlobalMacroID = id.(string)
	}
	return
}

// GlobalMacrosUpdate Wrapper for usermacro.updateglobal
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usermacro/updateglobal
func (api *API) GlobalMacrosUpdate(macros GlobalMacros) (err error) {
	_, err = api.CallWithError("usermacro.updateglobal", macros)
	return
}

// GlobalMacrosDeleteByIds Wrapper for usermacro.deleteglobal
// https://www.zabbix.com/documentation/7.0/manual/api/reference/usermacro/deleteglobal
func (api *API) GlobalMacrosDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("usermacro.deleteglobal", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	macroids := result["globalmacroids"].([]interface{})
	if len(ids) != len(macroids) {
		err = &ExpectedMore{len(ids), len(macroids)}
	}
	return
}
