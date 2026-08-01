package zabbix

// ValueMapMappingType numeric type for valuemap mapping
type ValueMapMappingType string

const (
	ValueMapMappingEqual          ValueMapMappingType = "0"
	ValueMapMappingGreaterOrEqual ValueMapMappingType = "1"
	ValueMapMappingLessOrEqual    ValueMapMappingType = "2"
	ValueMapMappingInRange        ValueMapMappingType = "3"
	ValueMapMappingRegexp         ValueMapMappingType = "4"
	ValueMapMappingDefault        ValueMapMappingType = "5"
)

// ValueMapMapping represents a single mapping entry in a value map
type ValueMapMapping struct {
	Type     ValueMapMappingType `json:"type"`
	Value    string              `json:"value"`
	Newvalue string              `json:"newvalue"`
}

// ValueMapMappings is an array of ValueMapMapping
type ValueMapMappings []ValueMapMapping

// ValueMap represents a Zabbix value map object
// https://www.zabbix.com/documentation/7.0/manual/api/reference/valuemap/object
type ValueMap struct {
	ValueMapID string           `json:"valuemapid,omitempty"`
	HostID     string           `json:"hostid,omitempty"`
	Name       string           `json:"name"`
	Mappings   ValueMapMappings `json:"mappings,omitempty"`
	UUID       string           `json:"uuid,omitempty"`
}

// ValueMaps is an array of ValueMap
type ValueMaps []ValueMap

// ValueMapsGet Wrapper for valuemap.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/valuemap/get
func (api *API) ValueMapsGet(params Params) (res ValueMaps, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("valuemap.get", params, &res)
	return
}

// ValueMapGetByID Gets a value map by ID only if there is exactly 1 matching.
func (api *API) ValueMapGetByID(id string) (res *ValueMap, err error) {
	vms, err := api.ValueMapsGet(Params{"valuemapids": id, "selectMappings": "extend"})
	if err != nil {
		return
	}

	if len(vms) == 1 {
		res = &vms[0]
	} else {
		e := ExpectedOneResult(len(vms))
		err = &e
	}
	return
}

// ValueMapsCreate Wrapper for valuemap.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/valuemap/create
func (api *API) ValueMapsCreate(valuemaps ValueMaps) (err error) {
	response, err := api.CallWithError("valuemap.create", valuemaps)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["valuemapids"].([]interface{})
	for i, id := range ids {
		valuemaps[i].ValueMapID = id.(string)
	}
	return
}

// ValueMapsUpdate Wrapper for valuemap.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/valuemap/update
func (api *API) ValueMapsUpdate(valuemaps ValueMaps) (err error) {
	_, err = api.CallWithError("valuemap.update", valuemaps)
	return
}

// ValueMapsDeleteByIds Wrapper for valuemap.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/valuemap/delete
func (api *API) ValueMapsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("valuemap.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	vmids := result["valuemapids"].([]interface{})
	if len(ids) != len(vmids) {
		err = &ExpectedMore{len(ids), len(vmids)}
	}
	return
}
