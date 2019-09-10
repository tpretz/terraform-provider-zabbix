package zabbix

import (
	"github.com/AlekSi/reflector"
)

// Template represent Zabbix Template type returned from Zabbix API
// https://www.zabbix.com/documentation/3.2/manual/api/reference/template/object
type Template struct {
	TemplateID  string     `json:"templateid,omitempty"`
	Host        string     `json:"host"`
	Description string     `json:"description,omitempty"`
	Name        string     `json:"name,omitempty"`
	Groups      HostGroups `json:"groups"`
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
	return
}

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
