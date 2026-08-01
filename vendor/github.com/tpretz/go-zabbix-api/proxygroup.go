package zabbix

// ProxyGroup represents a Zabbix 7.0 proxy group object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/proxygroup/object
type ProxyGroup struct {
	ProxyGroupID  string `json:"proxy_groupid,omitempty"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	FailoverDelay string `json:"failover_delay,omitempty"`
	MinOnline     string `json:"min_online,omitempty"`

	// read-only
	State string `json:"state,omitempty"`
}

// ProxyGroups is an array of ProxyGroup
type ProxyGroups []ProxyGroup

// ProxyGroupsGet Wrapper for proxygroup.get
func (api *API) ProxyGroupsGet(params Params) (res ProxyGroups, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("proxygroup.get", params, &res)
	return
}

// ProxyGroupGetByID Gets a proxy group by ID only if there is exactly 1 matching result.
func (api *API) ProxyGroupGetByID(id string) (res *ProxyGroup, err error) {
	groups, err := api.ProxyGroupsGet(Params{"proxy_groupids": id})
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

// ProxyGroupsCreate Wrapper for proxygroup.create
func (api *API) ProxyGroupsCreate(proxyGroups ProxyGroups) (err error) {
	response, err := api.CallWithError("proxygroup.create", proxyGroups)
	if err != nil {
		return
	}
	result := response.Result.(map[string]interface{})
	ids := result["proxy_groupids"].([]interface{})
	for i, id := range ids {
		proxyGroups[i].ProxyGroupID = id.(string)
	}
	return
}

// ProxyGroupsUpdate Wrapper for proxygroup.update
func (api *API) ProxyGroupsUpdate(proxyGroups ProxyGroups) (err error) {
	_, err = api.CallWithError("proxygroup.update", proxyGroups)
	return
}

// ProxyGroupsDeleteByIds Wrapper for proxygroup.delete
func (api *API) ProxyGroupsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("proxygroup.delete", ids)
	if err != nil {
		return
	}
	result := response.Result.(map[string]interface{})
	deletedIds := result["proxy_groupids"].([]interface{})
	if len(ids) != len(deletedIds) {
		err = &ExpectedMore{len(ids), len(deletedIds)}
	}
	return
}
