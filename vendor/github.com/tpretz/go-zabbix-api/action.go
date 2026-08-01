package zabbix

// Action represents a Zabbix action object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/action/object
type Action struct {
	ActionID string `json:"actionid,omitempty"`
	Name     string `json:"name"`
	// immutable after create; cleared on update, so omitempty is required or
	// Zabbix receives "" and reports "an integer is expected"
	EventSource        string            `json:"eventsource,omitempty"`
	Status             string            `json:"status,omitempty"`
	EscPeriod          string            `json:"esc_period,omitempty"`
	PauseSuppressed    string            `json:"pause_suppressed,omitempty"`
	NotifyIfCanceled   string            `json:"notify_if_canceled,omitempty"`
	PauseSymptoms      string            `json:"pause_symptoms,omitempty"`
	Filter             *ActionFilter     `json:"filter,omitempty"`
	Operations         []ActionOperation `json:"operations,omitempty"`
	RecoveryOperations []ActionOperation `json:"recovery_operations,omitempty"`
	UpdateOperations   []ActionOperation `json:"update_operations,omitempty"`
}

// ActionFilter represents an action filter.
type ActionFilter struct {
	EvalType    string            `json:"evaltype"`
	Formula     string            `json:"formula,omitempty"`
	EvalFormula string            `json:"eval_formula,omitempty"`
	Conditions  []ActionCondition `json:"conditions"`
}

// ActionCondition represents a condition in an action filter.
type ActionCondition struct {
	ConditionType string `json:"conditiontype"`
	Operator      string `json:"operator,omitempty"`
	Value         string `json:"value"`
	Value2        string `json:"value2,omitempty"`
	FormulaID     string `json:"formulaid,omitempty"`
}

// ActionOperation represents an operation in an action.
type ActionOperation struct {
	OperationID   string               `json:"operationid,omitempty"`
	ActionID      string               `json:"actionid,omitempty"`
	OperationType string               `json:"operationtype"`
	EscPeriod     string               `json:"esc_period,omitempty"`
	EscStepFrom   string               `json:"esc_step_from,omitempty"`
	EscStepTo     string               `json:"esc_step_to,omitempty"`
	EvalType      string               `json:"evaltype,omitempty"`
	OpConditions  []interface{}        `json:"opconditions,omitempty"`
	OpMessage     *ActionOpMessage     `json:"opmessage,omitempty"`
	OpMessageGrp  []ActionOpMessageGrp `json:"opmessage_grp,omitempty"`
	OpMessageUsr  []ActionOpMessageUsr `json:"opmessage_usr,omitempty"`
	OpCommand     *ActionOpCommand     `json:"opcommand,omitempty"`
	OpCommandHst  []ActionOpCommandHst `json:"opcommand_hst,omitempty"`
	OpCommandGrp  []ActionOpCommandGrp `json:"opcommand_grp,omitempty"`
	OpGroup       []ActionOpGroup      `json:"opgroup,omitempty"`
	OpTemplate    []ActionOpTemplate   `json:"optemplate,omitempty"`
	OpInventory   *ActionOpInventory   `json:"opinventory,omitempty"`
}

// ActionOpMessage represents message settings for a message operation.
type ActionOpMessage struct {
	DefaultMsg  string `json:"default_msg,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Message     string `json:"message,omitempty"`
	MediatypeID string `json:"mediatypeid,omitempty"`
}

// ActionOpMessageGrp represents a user group to send the message to.
type ActionOpMessageGrp struct {
	UsrGrpID string `json:"usrgrpid"`
}

// ActionOpMessageUsr represents a user to send the message to.
type ActionOpMessageUsr struct {
	UserID string `json:"userid"`
}

// ActionOpCommand represents a remote command.
type ActionOpCommand struct {
	ScriptID string `json:"scriptid"`
}

// ActionOpCommandHst represents a host to run the remote command on.
type ActionOpCommandHst struct {
	OpCommandHstID string `json:"opcommand_hstid,omitempty"`
	HostID         string `json:"hostid"`
}

// ActionOpCommandGrp represents a host group to run the remote command on.
type ActionOpCommandGrp struct {
	OpCommandGrpID string `json:"opcommand_grpid,omitempty"`
	GroupID        string `json:"groupid"`
}

// ActionOpGroup represents a host group to add the host to.
type ActionOpGroup struct {
	GroupID string `json:"groupid"`
}

// ActionOpTemplate represents a template to link.
type ActionOpTemplate struct {
	TemplateID string `json:"templateid"`
}

// ActionOpInventory represents inventory mode setting.
type ActionOpInventory struct {
	InventoryMode string `json:"inventory_mode"`
}

// Actions is an array of Action
type Actions []Action

// ActionsGet Wrapper for action.get
func (api *API) ActionsGet(params Params) (res Actions, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("action.get", params, &res)
	return
}

// ActionGetByID Gets an action by ID only if there is exactly 1 matching action.
func (api *API) ActionGetByID(id string) (res *Action, err error) {
	actions, err := api.ActionsGet(Params{
		"actionids":                id,
		"selectFilter":             "extend",
		"selectOperations":         "extend",
		"selectRecoveryOperations": "extend",
		"selectUpdateOperations":   "extend",
	})
	if err != nil {
		return
	}

	if len(actions) == 1 {
		res = &actions[0]
	} else {
		e := ExpectedOneResult(len(actions))
		err = &e
	}
	return
}

// ActionsCreate Wrapper for action.create
func (api *API) ActionsCreate(actions Actions) (err error) {
	response, err := api.CallWithError("action.create", actions)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["actionids"].([]interface{})
	for i, id := range ids {
		actions[i].ActionID = id.(string)
	}
	return
}

// ActionsUpdate Wrapper for action.update
func (api *API) ActionsUpdate(actions Actions) (err error) {
	_, err = api.CallWithError("action.update", actions)
	return
}

// ActionsDeleteByIds Wrapper for action.delete
func (api *API) ActionsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("action.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	actionids := result["actionids"].([]interface{})
	if len(ids) != len(actionids) {
		err = &ExpectedMore{len(ids), len(actionids)}
	}
	return
}
