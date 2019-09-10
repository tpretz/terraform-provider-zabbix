package zabbix

import (
	"github.com/AlekSi/reflector"
)

type (
	// InternalType (readonly) Whether the group is used internally by the system. An internal group cannot be deleted.
	// see "internal" in https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/object
	InternalType int
)

const (
	// NotInternal (default) not internal
	NotInternal InternalType = 0
	// Internal internal
	Internal InternalType = 1
)

// HostGroup represent Zabbix host group object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/object
type HostGroup struct {
	GroupID  string       `json:"groupid,omitempty"`
	Name     string       `json:"name"`
	Internal InternalType `json:"internal,omitempty"`
}

// HostGroups is an array of HostGroup
type HostGroups []HostGroup

// HostGroupID represent Zabbix GroupID
type HostGroupID struct {
	GroupID string `json:"groupid"`
}

// HostGroupIDs is an array of HostGroupId
type HostGroupIDs []HostGroupID

// HostGroupsGet Wrapper for hostgroup.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/get
func (api *API) HostGroupsGet(params Params) (res HostGroups, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	response, err := api.CallWithError("hostgroup.get", params)
	if err != nil {
		return
	}

	reflector.MapsToStructs2(response.Result.([]interface{}), &res, reflector.Strconv, "json")
	return
}

// HostGroupGetByID Gets host group by Id only if there is exactly 1 matching host group.
func (api *API) HostGroupGetByID(id string) (res *HostGroup, err error) {
	groups, err := api.HostGroupsGet(Params{"groupids": id})
	if err != nil {
		return
	}

	if len(groups) == 1 {
		res = &groups[0]
	} else {
		e := ExpectedOneResult(len(groups))
		err = &e
	}
	return
}

// HostGroupsCreate Wrapper for hostgroup.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/create
func (api *API) HostGroupsCreate(hostGroups HostGroups) (err error) {
	response, err := api.CallWithError("hostgroup.create", hostGroups)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["groupids"].([]interface{})
	for i, id := range groupids {
		hostGroups[i].GroupID = id.(string)
	}
	return
}

// HostGroupsUpdate Wrapper for hostgroup.update
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/update
func (api *API) HostGroupsUpdate(hostGroups HostGroups) (err error) {
	_, err = api.CallWithError("hostgroup.update", hostGroups)
	return
}

// HostGroupsDelete Wrapper for hostgroup.delete
// Cleans GroupId in all hostGroups elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/delete
func (api *API) HostGroupsDelete(hostGroups HostGroups) (err error) {
	ids := make([]string, len(hostGroups))
	for i, group := range hostGroups {
		ids[i] = group.GroupID
	}

	err = api.HostGroupsDeleteByIds(ids)
	if err == nil {
		for i := range hostGroups {
			hostGroups[i].GroupID = ""
		}
	}
	return
}

// HostGroupsDeleteByIds Wrapper for hostgroup.delete
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/delete
func (api *API) HostGroupsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("hostgroup.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["groupids"].([]interface{})
	if len(ids) != len(groupids) {
		err = &ExpectedMore{len(ids), len(groupids)}
	}
	return
}
