package zabbix

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

type Tag struct {
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

type Tags []Tag

type TriggerID struct {
	TriggerID string `json:"triggerid"`
}

// TemplateIDs is an Array of TemplateID structs.
type TriggerIDs []TriggerID

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
	//Value ValueType `json:""`

	Opdata             string `json:"opdata,omitempty"`
	Type               int    `json:"type"`
	Url                string `json:"url,omitempty"`
	RecoveryMode       int    `json:"recovery_mode"`
	RecoveryExpression string `json:"recovery_expression,omitempty"`
	CorrelationMode    int    `json:"correlation_mode"`
	CorrelationTag     string `json:"correlation_tag,omitempty"`
	ManualClose        int    `json:"manual_close"`

	Priority     SeverityType     `json:"priority,string"`
	Status       StatusType       `json:"status,string"`
	Dependencies TriggerIDs       `json:"dependencies,omitempty"`
	Functions    TriggerFunctions `json:"functions,omitempty"`
	// Items contained by the trigger in the items property.
	ContainedItems Items `json:"items,omitempty"`
	// Hosts that the trigger belongs to in the hosts property.
	ParentHosts Hosts `json:"hosts,omitempty"`
	Tags        Tags  `json:"tags,omitempty"`
}

// Triggers is an array of Trigger
type Triggers []Trigger

// TriggersGet Wrapper for trigger.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/trigger/get
func (api *API) TriggersGet(params Params) (res Triggers, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("trigger.get", params, &res)
	return
}
func (api *API) ProtoTriggersGet(params Params) (res Triggers, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("triggerprototype.get", params, &res)
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
func (api *API) ProtoTriggerGetByID(id string) (res *Trigger, err error) {
	triggers, err := api.ProtoTriggersGet(Params{"triggerids": id})
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
func (api *API) ProtoTriggersCreate(triggers Triggers) (err error) {
	response, err := api.CallWithError("triggerprototype.create", triggers)
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
func (api *API) ProtoTriggersUpdate(triggers Triggers) (err error) {
	_, err = api.CallWithError("triggerprototype.update", triggers)
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
func (api *API) ProtoTriggersDelete(triggers Triggers) (err error) {
	ids := make([]string, len(triggers))
	for i, trigger := range triggers {
		ids[i] = trigger.TriggerID
	}

	err = api.ProtoTriggersDeleteByIds(ids)
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
	deleteIds, err := api.TriggersDeleteIDs(ids)
	if err != nil {
		return
	}
	l := len(deleteIds)
	if len(ids) != l {
		err = &ExpectedMore{len(ids), l}
	}
	return
}
func (api *API) ProtoTriggersDeleteByIds(ids []string) (err error) {
	deleteIds, err := api.ProtoTriggersDeleteIDs(ids)
	if err != nil {
		return
	}
	l := len(deleteIds)
	if len(ids) != l {
		err = &ExpectedMore{len(ids), l}
	}
	return
}

// TriggersDeleteIDs Wrapper for trigger.delete
// return the id of the deleted trigger
func (api *API) TriggersDeleteIDs(ids []string) (triggerids []interface{}, err error) {
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
func (api *API) ProtoTriggersDeleteIDs(ids []string) (triggerids []interface{}, err error) {
	response, err := api.CallWithError("triggerprototype.delete", ids)
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
