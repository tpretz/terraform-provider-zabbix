package zabbix

import "encoding/json"

type (
	// AvailableType (readonly) Availability of Zabbix agent
	// see "available" in: https://www.zabbix.com/documentation/3.2/manual/api/reference/host/object
	AvailableType int

	// StatusType Status and function of the host.
	// see "status" in:	https://www.zabbix.com/documentation/3.2/manual/api/reference/host/object
	StatusType int
)

const (
	// Unknown (default)
	Unknown AvailableType = 0
	// Available host is available
	Available AvailableType = 1
	// Unavailable host is unavailable
	Unavailable AvailableType = 2
)

const (
	// Monitored monitored host(default)
	Monitored StatusType = 0
	// Unmonitored unmonitored host
	Unmonitored StatusType = 1
)

// Host represent Zabbix host object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/object
type Host struct {
	HostID     string        `json:"hostid,omitempty"`
	Host       string        `json:"host"`
	Available  AvailableType `json:"available,string"`
	Error      string        `json:"error"`
	Name       string        `json:"name"`
	Status     StatusType    `json:"status,string"`
	UserMacros Macros        `json:"macros,omitempty"`

	RawInventory json.RawMessage `json:"inventory,omitempty"`
	Inventory    *Inventory      `json:"-"`

	// Fields below used only when creating hosts
	GroupIds         HostGroupIDs   `json:"groups,omitempty"`
	Interfaces       HostInterfaces `json:"interfaces,omitempty"`
	TemplateIDs      TemplateIDs    `json:"templates,omitempty"`
	TemplateIDsClear TemplateIDs    `json:"templates_clear,omitempty"`
	// templates are read back from this one
	ParentTemplateIDs TemplateIDs `json:"parentTemplates,omitempty"`
	ProxyID           string      `json:"proxy_hostid,omitempty"`
}

// Hosts is an array of Host
type Hosts []Host

// HostsGet Wrapper for host.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/get
func (api *API) HostsGet(params Params) (res Hosts, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("host.get", params, &res)

	// fix up host interface details if present
	for i := 0; i < len(res); i++ {
		h := res[i]
		for j := 0; j < len(h.Interfaces); j++ {
			in := h.Interfaces[j]
			res[i].Interfaces[j].Details = nil
			if len(in.RawDetails) == 0 {
				continue
			}

			asStr := string(in.RawDetails)
			if asStr == "[]" {
				continue
			}

			out := HostInterfaceDetail{}
			// assume singular, if api changes, this will fault
			err := json.Unmarshal(in.RawDetails, &out)
			if err != nil {
				api.printf("got error during unmarshal %s", err)
				panic(err)
			}
			res[i].Interfaces[j].Details = &out
		}

		// fix up host inventory if present
		if len(h.RawInventory) == 0 {
			continue
		}

		// if its an empty array
		asStr := string(h.RawInventory)
		if asStr == "[]" {
			continue
		}

		// lets unbox
		var inv Inventory
		if err := json.Unmarshal(h.RawInventory, &inv); err != nil {
			api.printf("got error during unmarshal %s", err)
			panic(err)
		}
		h.Inventory = &inv
	}

	return
}

// HostsGetByHostGroupIds Gets hosts by host group Ids.
func (api *API) HostsGetByHostGroupIds(ids []string) (res Hosts, err error) {
	return api.HostsGet(Params{"groupids": ids})
}

// HostsGetByHostGroups Gets hosts by host groups.
func (api *API) HostsGetByHostGroups(hostGroups HostGroups) (res Hosts, err error) {
	ids := make([]string, len(hostGroups))
	for i, id := range hostGroups {
		ids[i] = id.GroupID
	}
	return api.HostsGetByHostGroupIds(ids)
}

// HostGetByID Gets host by Id only if there is exactly 1 matching host.
func (api *API) HostGetByID(id string) (res *Host, err error) {
	hosts, err := api.HostsGet(Params{"hostids": id})
	if err != nil {
		return
	}

	if len(hosts) == 1 {
		res = &hosts[0]
	} else {
		e := ExpectedOneResult(len(hosts))
		err = &e
	}
	return
}

// HostGetByHost Gets host by Host only if there is exactly 1 matching host.
func (api *API) HostGetByHost(host string) (res *Host, err error) {
	hosts, err := api.HostsGet(Params{"filter": map[string]string{"host": host}})
	if err != nil {
		return
	}

	if len(hosts) == 1 {
		res = &hosts[0]
	} else {
		e := ExpectedOneResult(len(hosts))
		err = &e
	}
	return
}

// handle manual marshal
func prepHosts(hosts Hosts) {
	for i := 0; i < len(hosts); i++ {
		h := hosts[i]
		for j := 0; j < len(h.Interfaces); j++ {
			in := h.Interfaces[j]

			if in.Details == nil {
				continue
			}

			asB, _ := json.Marshal(in.Details)
			hosts[i].Interfaces[j].RawDetails = json.RawMessage(asB)
		}
		if h.Inventory != nil {
			asB, _ := json.Marshal(h.Inventory)
			hosts[i].RawInventory = json.RawMessage(asB)
		}
	}
}

// HostsCreate Wrapper for host.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/create
func (api *API) HostsCreate(hosts Hosts) (err error) {
	prepHosts(hosts)
	response, err := api.CallWithError("host.create", hosts)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	hostids := result["hostids"].([]interface{})
	for i, id := range hostids {
		hosts[i].HostID = id.(string)
	}
	return
}

// HostsUpdate Wrapper for host.update
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/update
func (api *API) HostsUpdate(hosts Hosts) (err error) {
	prepHosts(hosts)
	_, err = api.CallWithError("host.update", hosts)
	return
}

// HostsDelete Wrapper for host.delete
// Cleans HostId in all hosts elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/delete
func (api *API) HostsDelete(hosts Hosts) (err error) {
	ids := make([]string, len(hosts))
	for i, host := range hosts {
		ids[i] = host.HostID
	}

	err = api.HostsDeleteByIds(ids)
	if err == nil {
		for i := range hosts {
			hosts[i].HostID = ""
		}
	}
	return
}

// HostsDeleteByIds Wrapper for host.delete
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/delete
func (api *API) HostsDeleteByIds(ids []string) (err error) {
	hostIds := make([]map[string]string, len(ids))
	for i, id := range ids {
		hostIds[i] = map[string]string{"hostid": id}
	}

	response, err := api.CallWithError("host.delete", hostIds)
	if err != nil {
		// Zabbix 2.4 uses new syntax only
		if e, ok := err.(*Error); ok && e.Code == -32500 {
			response, err = api.CallWithError("host.delete", ids)
		}
	}
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	hostids := result["hostids"].([]interface{})
	if len(ids) != len(hostids) {
		err = &ExpectedMore{len(ids), len(hostids)}
	}
	return
}
