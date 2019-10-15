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

// TriggerFunction The function objects represents the functions used in the trigger expression
type TriggerFunction struct {
	FunctionID string `json:"functionid"`
	ItemID     string `json:"itemid"`
	Function   string `json:"function"`
	Parameter  string `json:"parameter"`
}

// TriggerFunctions is an array of TriggerFunction
type TriggerFunctions []TriggerFunction

// Trigger represent Zabbix trigger object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/object
type Trigger struct {
	TriggerID   string `json:"triggerid,omitempty"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
	Comments    string `json:"comments"`
	//TemplateId  string    `json:"templateid"`
	Value ValueType `json:""`

	Priority     SeverityType     `json:"priority"`
	Status       StatusType       `json:"status"`
	Dependencies Triggers         `json:"dependencies,omitempty"`
	Functions    TriggerFunctions `json:"functions,omitempty"`
	// Items contained by the trigger in the items property.
	ContainedItems Items `json:"items,omitempty"`
	// Hosts that the trigger belongs to in the hosts property.
	TriggerParent Hosts `json:"hosts,omitempty"`
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
	parseArray := response.Result.([]interface{})
	for i := range parseArray {
		parseResult := parseArray[i].(map[string]interface{})
		if _, present := parseResult["dependencies"]; present {
			reflector.MapsToStructs2(parseResult["dependencies"].([]interface{}), &(res[i].Dependencies), reflector.Strconv, "json")
		}
		if _, present := parseResult["functions"]; present {
			reflector.MapsToStructs2(parseResult["functions"].([]interface{}), &(res[i].Functions), reflector.Strconv, "json")
		}
		if _, present := parseResult["items"]; present {
			reflector.MapsToStructs2(parseResult["items"].([]interface{}), &(res[i].ContainedItems), reflector.Strconv, "json")
		}
		if _, present := parseResult["hosts"]; present {
			reflector.MapsToStructs2(parseResult["hosts"].([]interface{}), &(res[i].TriggerParent), reflector.Strconv, "json")
		}
	}
	return
}

// TriggerGetByID Gets trigger by Id only if there is exactly 1 matching host.
func (api *API) TriggerGetByID(id string) (res *Trigger, err error) {
	triggers, err := api.TriggersGet(Params{"triggerids": id})
	if err != nil {
		return
	}

	if len(triggers) != 1 {
		e := ExpectedOneResult(len(triggers))
		err = &e
		return
	}
	res = &triggers[0]
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

// TriggersUpdate Wrapper for trigger.update
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/update
func (api *API) TriggersUpdate(triggers Triggers) (err error) {
	_, err = api.CallWithError("trigger.update", triggers)
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

// TriggersDeleteNoError Wrapper for trigger.delete
// return the id of the deleted trigger
func (api *API) TriggersDeleteNoError(ids []string) (triggerids []interface{}, err error) {
	response, err := api.CallWithError("trigger.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	triggerids1, ok := result["triggerids"].([]interface{})
	if !ok {
		triggerids2 := result["triggerids"].(map[string]interface{})
		for _, id := range triggerids2 {
			triggerids = append(triggerids, id)
		}
	} else {
		triggerids = triggerids1
	}
	return
}
