package zabbix

import (
	"encoding/json"
	"fmt"
	"sort"
)

type (
	// ItemType type of the item
	ItemType int
	// ValueType type of information of the item
	ValueType int
	// DataType data type of the item
	DataType int
	// DeltaType value that will be stored
	DeltaType int
)

const (
	// Different item type, see :
	// - "type" in https://www.zabbix.com/documentation/3.2/manual/api/reference/item/object
	// - https://www.zabbix.com/documentation/3.2/manual/config/items/itemtypes

	// ZabbixAgent type
	ZabbixAgent ItemType = 0
	// ZabbixTrapper type
	ZabbixTrapper ItemType = 2
	// SimpleCheck type
	SimpleCheck ItemType = 3
	// ZabbixInternal type
	ZabbixInternal ItemType = 5
	// ZabbixAgentActive type
	ZabbixAgentActive ItemType = 7
	// WebItem type
	WebItem ItemType = 9
	// ExternalCheck type
	ExternalCheck ItemType = 10
	// DatabaseMonitor type
	DatabaseMonitor ItemType = 11
	//IPMIAgent type
	IPMIAgent ItemType = 12
	// SSHAgent type
	SSHAgent ItemType = 13
	// TELNETAgent type
	TELNETAgent ItemType = 14
	// Calculated type
	Calculated ItemType = 15
	// JMXAgent type
	JMXAgent  ItemType = 16
	SNMPTrap  ItemType = 17
	Dependent ItemType = 18
	HTTPAgent ItemType = 19
	SNMPAgent ItemType = 20
	// Script type (5.4+)
	Script ItemType = 21
	// Browser type (7.0+)
	Browser ItemType = 22
)

const (
	// Type of information of the item
	// see "value_type" in https://www.zabbix.com/documentation/3.2/manual/api/reference/item/object

	// Float value
	Float ValueType = 0
	// Character value
	Character ValueType = 1
	// Log value
	Log ValueType = 2
	// Unsigned value
	Unsigned ValueType = 3
	// Text value
	Text ValueType = 4
)

const (
	// Data type of the item
	// see "data_type" in https://www.zabbix.com/documentation/3.2/manual/api/reference/item/object

	// Decimal data (default)
	Decimal DataType = 0
	// Octal data
	Octal DataType = 1
	// Hexadecimal data
	Hexadecimal DataType = 2
	// Boolean data
	Boolean DataType = 3
)

const (
	// Value that will be stored
	// see "delta" in https://www.zabbix.com/documentation/3.2/manual/api/reference/item/object

	// AsIs as is (default)
	AsIs DeltaType = 0
	// Speed speed per second
	Speed DeltaType = 1
	// Delta simple change
	Delta DeltaType = 2
)

type HttpHeaders map[string]string

// httpHeaderPair is the Zabbix >= 7.0 wire form of a single HTTP header.
type httpHeaderPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// marshalHttpHeaders renders headers in the shape the server expects. Zabbix
// 7.0 changed the "headers" property of items, item prototypes and discovery
// rules from a name-indexed object to an array of {name, value} objects, and
// rejects the old shape with `an array is expected`.
func (api *API) marshalHttpHeaders(h HttpHeaders) json.RawMessage {
	if api.Config.Version >= V70 {
		// sorted so the request is stable and diffs stay readable
		names := make([]string, 0, len(h))
		for k := range h {
			names = append(names, k)
		}
		sort.Strings(names)
		arr := make([]httpHeaderPair, 0, len(names))
		for _, k := range names {
			arr = append(arr, httpHeaderPair{Name: k, Value: h[k]})
		}
		b, _ := json.Marshal(arr)
		return json.RawMessage(b)
	}
	b, _ := json.Marshal(map[string]string(h))
	return json.RawMessage(b)
}

// unmarshalHttpHeaders accepts either wire shape, so a client talking to a
// mixed-version estate does not need to know which it will get.
func (api *API) unmarshalHttpHeaders(raw json.RawMessage) HttpHeaders {
	out := HttpHeaders{}
	if len(raw) == 0 {
		return out
	}
	asStr := string(raw)
	if asStr == "[]" || asStr == "{}" || asStr == "null" {
		return out
	}

	var arr []httpHeaderPair
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, p := range arr {
			out[p.Name] = p.Value
		}
		return out
	}

	obj := map[string]string{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		api.printf("got error during unmarshal %s", err)
		panic(err)
	}
	for k, v := range obj {
		out[k] = v
	}
	return out
}

// PreprocNotSupported is the "check for not supported value" preprocessing step.
const PreprocNotSupported = "26"

// preprocNotSupportedAnyError is the params value that step 26 takes from
// Zabbix 7.0 on to mean "any error". Before 7.0 the step took no parameters at
// all and the server rejected a non-empty params.
const preprocNotSupportedAnyError = "-1"

// prepPreprocessors adapts preprocessing steps to the target version. Zabbix
// 7.0 gave "check for not supported value" a mandatory params value.
func (api *API) prepPreprocessors(p Preprocessors) {
	if api.Config.Version < V70 {
		return
	}
	for i := range p {
		if p[i].Type == PreprocNotSupported && p[i].Params == "" {
			p[i].Params = preprocNotSupportedAnyError
		}
	}
}

// readPreprocessors is the inverse of prepPreprocessors, so a configuration
// that leaves step 26 without params round-trips cleanly on 7.0+.
func (api *API) readPreprocessors(p Preprocessors) {
	if api.Config.Version < V70 {
		return
	}
	for i := range p {
		if p[i].Type == PreprocNotSupported && p[i].Params == preprocNotSupportedAnyError {
			p[i].Params = ""
		}
	}
}

// Item represent Zabbix item object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/object
type Item struct {
	ItemID string `json:"itemid,omitempty"`
	Delay  string `json:"delay"`
	// hostid is create-only from Zabbix 7.0 on — item.update and
	// itemprototype.update reject it as an unexpected parameter, so
	// prepItemsUpdate strips it.
	HostID      string    `json:"hostid,omitempty"`
	InterfaceID string    `json:"interfaceid,omitempty"`
	Key         string    `json:"key_"`
	Name        string    `json:"name"`
	Type        ItemType  `json:"type,string"`
	ValueType   ValueType `json:"value_type,string"`
	// data_type and delta were dropped from the item object in Zabbix 3.4 and
	// are rejected outright by item.create/update from 7.0 on, so they are not
	// serialised at all.
	DataType     DataType  `json:"-"`
	Delta        DeltaType `json:"-"`
	Description  string    `json:"description"`
	Error        string    `json:"error,omitempty"`
	History      string    `json:"history,omitempty"`
	Trends       string    `json:"trends,omitempty"`
	TrapperHosts string    `json:"trapper_hosts,omitempty"`
	Params       string    `json:"params,omitempty"`

	// read-only, only populated by selectHosts; omitempty keeps it off the
	// write path, where 7.0+ rejects it as an unexpected property
	ItemParent Hosts `json:"hosts,omitempty"`

	Preprocessors Preprocessors `json:"preprocessing,omitempty"`

	// HTTP Agent Fields
	Url             string          `json:"url,omitempty"`
	RequestMethod   string          `json:"request_method,omitempty"`
	PostType        string          `json:"post_type,omitempty"`
	RetrieveMode    string          `json:"retrieve_mode,omitempty"`
	Posts           string          `json:"posts,omitempty"`
	StatusCodes     string          `json:"status_codes,omitempty"`
	Timeout         string          `json:"timeout,omitempty"`
	VerifyHost      string          `json:"verify_host,omitempty"`
	VerifyPeer      string          `json:"verify_peer,omitempty"`
	AuthType        string          `json:"authtype,omitempty"`
	Username        string          `json:"username,omitempty"`
	Password        string          `json:"password,omitempty"`
	Headers         HttpHeaders     `json:"-"`
	RawHeaders      json.RawMessage `json:"headers,omitempty"`
	Proxy           string          `json:"http_proxy,omitempty"`
	FollowRedirects string          `json:"follow_redirects,omitempty"`

	// SNMP Fields
	SNMPOid string `json:"snmp_oid,omitempty"`

	// Dependent Fields
	MasterItemID string `json:"master_itemid,omitempty"`

	// Prototype. discoveryRule is read-only (selectDiscoveryRule) — note the
	// tag was "omitEmpty", which encoding/json does not recognise, so it used
	// to be serialised as null on every create/update.
	RuleID        string   `json:"ruleid,omitempty"`
	DiscoveryRule *LLDRule `json:"discoveryRule,omitempty"`

	Tags Tags `json:"tags,omitempty"`
}

type Preprocessors []Preprocessor

type Preprocessor struct {
	Type               string `json:"type,omitempty"`
	Params             string `json:"params"`
	ErrorHandler       string `json:"error_handler,omitempty"`
	ErrorHandlerParams string `json:"error_handler_params"`
}

// Items is an array of Item
type Items []Item

// ByKey Converts slice to map by key. Panics if there are duplicate keys.
func (items Items) ByKey() (res map[string]Item) {
	res = make(map[string]Item, len(items))
	for _, i := range items {
		_, present := res[i.Key]
		if present {
			panic(fmt.Errorf("Duplicate key %s", i.Key))
		}
		res[i.Key] = i
	}
	return
}

// ItemsGet Wrapper for item.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/get
func (api *API) ItemsGet(params Params) (res Items, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("item.get", params, &res)
	api.itemsHeadersUnmarshal(res)
	return
}
func (api *API) ProtoItemsGet(params Params) (res Items, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("itemprototype.get", params, &res)
	api.itemsHeadersUnmarshal(res)
	return
}

func (api *API) itemsHeadersUnmarshal(item Items) {
	for i := 0; i < len(item); i++ {
		h := item[i]

		item[i].Headers = api.unmarshalHttpHeaders(h.RawHeaders)
		api.readPreprocessors(item[i].Preprocessors)
	}
}

func (api *API) prepItems(item Items) {
	for i := 0; i < len(item); i++ {
		h := item[i]

		// read-only, never valid on the write path
		item[i].ItemParent = nil
		item[i].DiscoveryRule = nil

		api.prepPreprocessors(item[i].Preprocessors)

		if h.Headers == nil {
			continue
		}
		item[i].RawHeaders = api.marshalHttpHeaders(h.Headers)
	}
}

// prepItemsUpdate is prepItems plus the properties that are valid on create but
// not on update.
func (api *API) prepItemsUpdate(item Items) {
	api.prepItems(item)

	// "ruleid" names the discovery rule an item prototype is created under. It
	// is create-only: itemprototype.update rejects it outright on 7.2+, where
	// unknown request parameters became a hard error, which made every
	// zabbix_proto_item_* resource impossible to update there. Older versions
	// ignored it, so clearing it unconditionally is safe. Plain items never
	// carry a RuleID, so this is a no-op for item.update.
	for i := range item {
		item[i].RuleID = ""
	}

	if api.Config.Version < V70 {
		return
	}
	for i := range item {
		item[i].HostID = ""
	}
}

// ItemGetByID Gets item by Id only if there is exactly 1 matching host.
func (api *API) ItemGetByID(id string) (res *Item, err error) {
	items, err := api.ItemsGet(Params{"itemids": id})
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
func (api *API) ProtoItemGetByID(id string) (res *Item, err error) {
	items, err := api.ProtoItemsGet(Params{"itemids": id})
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
func (api *API) ItemsCreate(items Items) (err error) {
	api.prepItems(items)
	response, err := api.CallWithError("item.create", items)
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
func (api *API) ProtoItemsCreate(items Items) (err error) {
	api.prepItems(items)
	response, err := api.CallWithError("itemprototype.create", items)
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
func (api *API) ItemsUpdate(items Items) (err error) {
	api.prepItemsUpdate(items)
	_, err = api.CallWithError("item.update", items)
	return
}
func (api *API) ProtoItemsUpdate(items Items) (err error) {
	api.prepItemsUpdate(items)
	_, err = api.CallWithError("itemprototype.update", items)
	return
}

// ItemsDelete Wrapper for item.delete
// Cleans ItemId in all items elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/delete
func (api *API) ItemsDelete(items Items) (err error) {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ItemID
	}

	err = api.ItemsDeleteByIds(ids)
	if err == nil {
		for i := range items {
			items[i].ItemID = ""
		}
	}
	return
}
func (api *API) ProtoItemsDelete(items Items) (err error) {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ItemID
	}

	err = api.ProtoItemsDeleteByIds(ids)
	if err == nil {
		for i := range items {
			items[i].ItemID = ""
		}
	}
	return
}

// ItemsDeleteByIds Wrapper for item.delete
// https://www.zabbix.com/documentation/3.2/manual/api/reference/item/delete
func (api *API) ItemsDeleteByIds(ids []string) (err error) {
	deleteIds, err := api.ItemsDeleteIDs(ids)
	if err != nil {
		return
	}
	l := len(deleteIds)
	if len(ids) != l {
		err = &ExpectedMore{len(ids), l}
	}
	return
}
func (api *API) ProtoItemsDeleteByIds(ids []string) (err error) {
	deleteIds, err := api.ProtoItemsDeleteIDs(ids)
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
func (api *API) ItemsDeleteIDs(ids []string) (itemids []interface{}, err error) {
	response, err := api.CallWithError("item.delete", ids)
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
func (api *API) ProtoItemsDeleteIDs(ids []string) (itemids []interface{}, err error) {
	response, err := api.CallWithError("itemprototype.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	itemids1, ok := result["prototypeids"].([]interface{})
	if !ok {
		itemids2 := result["prototypeids"].(map[string]interface{})
		for _, id := range itemids2 {
			itemids = append(itemids, id)
		}
	} else {
		itemids = itemids1
	}
	return
}
