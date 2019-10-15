package zabbix

import (
	"github.com/AlekSi/reflector"
)

// Template represent Zabbix Template type returned from Zabbix API
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/object
type Template struct {
	TemplateID      string     `json:"templateid,omitempty"`
	Host            string     `json:"host"`
	Description     string     `json:"description,omitempty"`
	Name            string     `json:"name,omitempty"`
	Groups          HostGroups `json:"groups"`
	UserMacros      Macros     `json:"macros"`
	LinkedTemplates Templates  `json:"templates,omitempty"`
	TemplatesClear  Templates  `json:"templates_clear,omitempty"`
	LinkedHosts     []string   `json:"hosts,omitempty"`
}

// Templates is an Array of Template structs.
type Templates []Template

// TemplateID use with host creation
type TemplateID struct {
	TemplateID string `json:"templateid"`
}

// TemplateIDs is an Array of TemplateID structs.
type TemplateIDs []TemplateID

// TemplatesGet Wrapper for template.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/get
func (api *API) TemplatesGet(params Params) (res Templates, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	response, err := api.CallWithError("template.get", params)
	if err != nil {
		return
	}

	reflector.MapsToStructs2(response.Result.([]interface{}), &res, reflector.Strconv, "json")
	parseArray := response.Result.([]interface{})
	for i := range parseArray {
		parseResult := parseArray[i].(map[string]interface{})
		if _, present := parseResult["macros"]; present {
			reflector.MapsToStructs2(parseResult["macros"].([]interface{}), &(res[i].UserMacros), reflector.Strconv, "json")
		}
		if _, present := parseResult["templates"]; present {
			var templates Templates
			reflector.MapsToStructs2(parseResult["templates"].([]interface{}), &templates, reflector.Strconv, "json")
			for _, template := range templates {
				res[i].LinkedHosts = append(res[i].LinkedHosts, template.TemplateID)
			}
		}
		if _, present := parseResult["hosts"]; present {
			var hosts Hosts
			reflector.MapsToStructs2(parseResult["hosts"].([]interface{}), &hosts, reflector.Strconv, "json")
			for _, host := range hosts {
				res[i].LinkedHosts = append(res[i].LinkedHosts, host.HostID)
			}
		}
		if _, present := parseResult["parentTemplates"]; present {
			reflector.MapsToStructs2(parseResult["parentTemplates"].([]interface{}), &(res[i].LinkedTemplates), reflector.Strconv, "json")
		}
	}
	return
}

// TemplateGetByID Gets template by Id only if there is exactly 1 matching template.
func (api *API) TemplateGetByID(id string) (template *Template, err error) {
	templates, err := api.TemplatesGet(Params{"templateids": id})
	if err != nil {
		return
	}

	if len(templates) == 1 {
		template = &templates[0]
	} else {
		e := ExpectedOneResult(len(templates))
		err = &e
	}
	return
}

// TemplatesCreate Wrapper for template.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/create
func (api *API) TemplatesCreate(templates Templates) (err error) {
	response, err := api.CallWithError("template.create", templates)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	templateids := result["templateids"].([]interface{})
	for i, id := range templateids {
		templates[i].TemplateID = id.(string)
	}
	return
}

// TemplatesUpdate Wrapper for template.update
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/update
func (api *API) TemplatesUpdate(templates Templates) (err error) {
	_, err = api.CallWithError("template.update", templates)
	return
}

// TemplatesDelete Wrapper for template.delete
// Cleans ApplicationID in all apps elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/delete
func (api *API) TemplatesDelete(templates Templates) (err error) {
	templatesIds := make([]string, len(templates))
	for i, template := range templates {
		templatesIds[i] = template.TemplateID
	}

	err = api.TemplatesDeleteByIds(templatesIds)
	if err == nil {
		for i := range templates {
			templates[i].TemplateID = ""
		}
	}
	return
}

// TemplatesDeleteByIds Wrapper for template.delete
// Use template's id to delete the template
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/delete
func (api *API) TemplatesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("template.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	templateids := result["templateids"].([]interface{})
	if len(ids) != len(templateids) {
		err = &ExpectedMore{len(ids), len(templateids)}
	}
	return
}
