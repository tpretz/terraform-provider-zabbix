package zabbix

type LLDRuleFilterCondition struct {
	Macro     string `json:"macro"`
	Value     string `json:"value"`
	FormulaID string `json:"formulaid,omitempty"`
	Operator  string `json:"operator,omitempty"`
}

type LLDRuleFilterConditions []LLDRuleFilterCondition

type LLDRuleFilter struct {
	Conditions LLDRuleFilterConditions `json:"conditions`
}

// Item represent Zabbix lld object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/object
type LLDRule struct {
	ItemID      string   `json:"itemid,omitempty"`
	Delay       string   `json:"delay"`
	HostID      string   `json:"hostid"`
	InterfaceID string   `json:"interfaceid,omitempty"`
	Key         string   `json:"key_"`
	Name        string   `json:"name"`
	Type        ItemType `json:"type,string"`
	//	ValueType    ValueType `json:"value_type,string"`
	//	DataType     DataType  `json:"data_type,string"`
	//	Delta        DeltaType `json:"delta,string"`
	AuthType     string `json:"authtype,omitempty"`
	DelayFlex    string `json:"delay_flex,omitempty"`
	Description  string `json:"description"`
	Error        string `json:"error,omitempty"`
	IpmiSensor   string `json:"ipmi_sensor,omitempty"`
	LifeTime     string `json:"lifetime,omitempty"`
	Params       string `json:"params,omitempty"`
	PrivateKey   string `json:"privatekey,omitempty"`
	PublicKey    string `json:"publickey,omitempty"`
	Status       string `json:"status,omitempty"`
	TrapperHosts string `json:"trapper_hosts,omitempty"`

	// ssh / telnet
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Port     string `json:"port,omitempty"`

	// HTTP Agent Fields
	Url           string `json:"url,omitempty"`
	RequestMethod string `json:"request_method,omitempty"`
	AllowTraps    string `json:"allow_traps,omitempty"`
	PostType      string `json:"post_type,omitempty"`
	Posts         string `json:"posts,omitempty"`
	StatusCodes   string `json:"status_codes,omitempty"`
	Timeout       string `json:"timeout,omitempty"`
	VerifyHost    string `json:"verify_host,omitempty"`
	VerifyPeer    string `json:"verify_peer,omitempty"`

	// SNMP Fields
	SNMPOid              string `json:"snmp_oid,omitempty"`
	SNMPCommunity        string `json:"snmp_community,omitempty"`
	SNMPv3AuthPassphrase string `json:"snmpv3_authpassphrase,omitempty"`
	SNMPv3AuthProtocol   string `json:"snmpv3_authprotocol,omitempty"`
	SNMPv3ContextName    string `json:"snmpv3_contextname,omitempty"`
	SNMPv3PrivPasshrase  string `json:"snmpv3_privpassphrase,omitempty"`
	SNMPv3PrivProtocol   string `json:"snmpv3_privprotocol,omitempty"`
	SNMPv3SecurityLevel  string `json:"snmpv3_securitylevel,omitempty"`
	SNMPv3SecurityName   string `json:"snmpv3_securityname,omitempty"`

	Preprocessors Preprocessors `json:"preprocessing,omitempty"`
	Filter        LLDRuleFilter `json:"filter"`
}

// Items is an array of Item
type LLDRules []LLDRule

// ItemsGet Wrapper for item.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/get
func (api *API) LLDsGet(params Params) (res LLDRules, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("discoveryrule.get", params, &res)
	return
}

// ItemGetByID Gets item by Id only if there is exactly 1 matching host.
func (api *API) LLDGetByID(id string) (res *LLDRule, err error) {
	items, err := api.LLDsGet(Params{"itemids": id})
	if err != nil {
		return
	}

	if len(items) != 1 {
		e := ExpectedOneResult(len(items))
		err = &e
		return
	}
	res = &items[0]
	return
}

// ItemsCreate Wrapper for item.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/create
func (api *API) LLDsCreate(items LLDRules) (err error) {
	response, err := api.CallWithError("discoveryrule.create", items)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	itemids := result["itemids"].([]interface{})
	for i, id := range itemids {
		items[i].ItemID = id.(string)
	}
	return
}

// ItemsUpdate Wrapper for item.update
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/update
func (api *API) LLDsUpdate(items LLDRules) (err error) {
	_, err = api.CallWithError("discoveryrule.update", items)
	return
}

// ItemsDelete Wrapper for item.delete
// Cleans ItemId in all items elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/delete
func (api *API) LLDsDelete(items LLDRules) (err error) {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ItemID
	}

	err = api.LLDDeleteByIds(ids)
	if err == nil {
		for i := range items {
			items[i].ItemID = ""
		}
	}
	return
}

// ItemsDeleteByIds Wrapper for item.delete
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/delete
func (api *API) LLDDeleteByIds(ids []string) (err error) {
	deleteIds, err := api.LLDDeleteIDs(ids)
	if err != nil {
		return
	}
	l := len(deleteIds)
	if len(ids) != l {
		err = &ExpectedMore{len(ids), l}
	}
	return
}

// ItemsDeleteIDs Wrapper for item.delete
// Delete the item and return the id of the deleted item
func (api *API) LLDDeleteIDs(ids []string) (itemids []interface{}, err error) {
	response, err := api.CallWithError("discoveryrule.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	itemids1, ok := result["itemids"].([]interface{})
	if !ok {
		itemids2 := result["itemids"].(map[string]interface{})
		for _, id := range itemids2 {
			itemids = append(itemids, id)
		}
	} else {
		itemids = itemids1
	}
	return
}
