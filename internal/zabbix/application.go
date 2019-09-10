package zabbix

import (
	"github.com/AlekSi/reflector"
)

// Application represent Zabbix application object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/application/object
type Application struct {
	ApplicationID string `json:"applicationid,omitempty"`
	HostID        string `json:"hostid"`
	Name          string `json:"name"`
	TemplateID    string `json:"templateid,omitempty"`
}

// Applications is an array of Application
type Applications []Application

// ApplicationsGet Wrapper for application.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/application/get
func (api *API) ApplicationsGet(params Params) (res Applications, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	response, err := api.CallWithError("application.get", params)
	if err != nil {
		return
	}

	reflector.MapsToStructs2(response.Result.([]interface{}), &res, reflector.Strconv, "json")
	return
}

// ApplicationGetByID Gets application by Id only if there is exactly 1 matching application.
func (api *API) ApplicationGetByID(id string) (res *Application, err error) {
	apps, err := api.ApplicationsGet(Params{"applicationids": id})
	if err != nil {
		return
	}

	if len(apps) == 1 {
		res = &apps[0]
	} else {
		e := ExpectedOneResult(len(apps))
		err = &e
	}
	return
}

// ApplicationGetByHostIDAndName Gets application by host Id and name only if there is exactly 1 matching application.
func (api *API) ApplicationGetByHostIDAndName(hostID, name string) (res *Application, err error) {
	apps, err := api.ApplicationsGet(Params{"hostids": hostID, "filter": map[string]string{"name": name}})
	if err != nil {
		return
	}

	if len(apps) == 1 {
		res = &apps[0]
	} else {
		e := ExpectedOneResult(len(apps))
		err = &e
	}
	return
}

// ApplicationsCreate Wrapper for application.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/application/create
func (api *API) ApplicationsCreate(apps Applications) (err error) {
	response, err := api.CallWithError("application.create", apps)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	applicationids := result["applicationids"].([]interface{})
	for i, id := range applicationids {
		apps[i].ApplicationID = id.(string)
	}
	return
}

// ApplicationsDelete Wrapper for application.delete:
// Cleans ApplicationID in all apps elements if call succeed.
// https://www.zabbix.com/documentation/2.2/manual/appendix/api/application/delete
func (api *API) ApplicationsDelete(apps Applications) (err error) {
	ids := make([]string, len(apps))
	for i, app := range apps {
		ids[i] = app.ApplicationID
	}

	err = api.ApplicationsDeleteByIds(ids)
	if err == nil {
		for i := range apps {
			apps[i].ApplicationID = ""
		}
	}
	return
}

// ApplicationsDeleteByIds Wrapper for application.delete
// https://www.zabbix.com/documentation/2.2/manual/appendix/api/application/delete
func (api *API) ApplicationsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("application.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	applicationids := result["applicationids"].([]interface{})
	if len(ids) != len(applicationids) {
		err = &ExpectedMore{len(ids), len(applicationids)}
	}
	return
}
