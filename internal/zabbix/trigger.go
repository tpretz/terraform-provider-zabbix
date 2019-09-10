package zabbix

import (
	"github.com/AlekSi/reflector"
)

type (
	// SeverityType of a trigger
	// Zabbix severity see : https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/object
	SeverityType int
)

const (
	// Different severity see : https://www.zabbix.com/documentation/3.2/manual/config/triggers/severity

	// NotClassified is Not classified severity
	NotClassified SeverityType = 0
	// Information is Information severity
	Information SeverityType = 1
	// Warning is Warning severity
	Warning SeverityType = 2
	// Average is Average severity
	Average SeverityType = 3
	// High is high severity
	High SeverityType = 4
	// Critical is critical severity
	Critical SeverityType = 5
)

const (
	// Enabled trigger status enabled
	Enabled StatusType = 0
	// Disabled trigger status disabled
	Disabled StatusType = 1
)

const (
	// Trigger value see : https://www.zabbix.com/documentation/3.2/manual/config/triggers

	// OK trigger value ok
	OK ValueType = 0
	// Problem trigger value probleme
	Problem ValueType = 1
)

// Trigger represent Zabbix trigger object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/object
type Trigger struct {
	TriggerID   string `json:"triggerid,omitempty"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
	Comments    string `json:"comments"`
	//TemplateId  string    `json:"templateid"`
	Value ValueType `json:""`

	Priority SeverityType `json:"priority"`
	Status   StatusType   `json:"status"`
}

// Triggers is an array of Trigger
type Triggers []Trigger

// TriggersGet Wrapper for trigger.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/get
func (api *API) TriggersGet(params Params) (res Triggers, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	response, err := api.CallWithError("trigger.get", params)
	if err != nil {
		return
	}

	reflector.MapsToStructs2(response.Result.([]interface{}), &res, reflector.Strconv, "json")
	return
}

// TriggersCreate Wrapper for trigger.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/create
func (api *API) TriggersCreate(triggers Triggers) (err error) {
	response, err := api.CallWithError("trigger.create", triggers)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	triggerids := result["triggerids"].([]interface{})
	for i, id := range triggerids {
		triggers[i].TriggerID = id.(string)
	}
	return
}

// TriggersDelete Wrapper for trigger.delete
// Cleans ItemId in all triggers elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/delete
func (api *API) TriggersDelete(triggers Triggers) (err error) {
	ids := make([]string, len(triggers))
	for i, trigger := range triggers {
		ids[i] = trigger.TriggerID
	}

	err = api.TriggersDeleteByIds(ids)
	if err == nil {
		for i := range triggers {
			triggers[i].TriggerID = ""
		}
	}
	return
}

// TriggersDeleteByIds Wrapper for trigger.delete
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/delete
func (api *API) TriggersDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("trigger.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	triggerids1, ok := result["triggerids"].([]interface{})
	l := len(triggerids1)
	if !ok {
		// some versions actually return map there
		triggerids2 := result["triggerids"].(map[string]interface{})
		l = len(triggerids2)
	}
	if len(ids) != l {
		err = &ExpectedMore{len(ids), l}
	}
	return
}
