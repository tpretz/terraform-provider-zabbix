package zabbix

import "encoding/json"

type (
	// AvailableType (readonly) Availability of Zabbix agent
	// see "available" in: https://www.zabbix.com/documentation/3.2/manual/api/reference/host/object
	AvailableType int

	// StatusType Status and function of the host.
	// see "status" in:	https://www.zabbix.com/documentation/3.2/manual/api/reference/host/object
	StatusType int

	InventoryMode int

	// MonitoredByType is the Zabbix >= 7.0 "monitored_by" host property: what
	// carries out the host's monitoring.
	MonitoredByType string
)

const (
	// MonitoredByServer host is monitored by the Zabbix server (default)
	MonitoredByServer MonitoredByType = "0"
	// MonitoredByProxy host is monitored by a proxy
	MonitoredByProxy MonitoredByType = "1"
	// MonitoredByProxyGroup host is monitored by a proxy group
	MonitoredByProxyGroup MonitoredByType = "2"
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
	InventoryDisabled  InventoryMode = -1
	InventoryManual    InventoryMode = 0
	InventoryAutomatic InventoryMode = 1
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
	UserMacros Macros        `json:"macros"`

	RawInventory json.RawMessage `json:"inventory,omitempty"`
	Inventory    Inventory       `json:"-"`

	RawInventoryMode json.RawMessage `json:"inventory_mode,omitempty"`
	InventoryMode    InventoryMode   `json:"-"`

	// Fields below used only when creating hosts
	GroupIds         HostGroupIDs   `json:"groups,omitempty"`
	Interfaces       HostInterfaces `json:"interfaces,omitempty"`
	TemplateIDs      TemplateIDs    `json:"templates,omitempty"`
	TemplateIDsClear TemplateIDs    `json:"templates_clear,omitempty"`
	// templates are read back from this one
	ParentTemplateIDs TemplateIDs     `json:"parentTemplates,omitempty"`
	Tags              Tags            `json:"-"`
	RawTags           json.RawMessage `json:"tags,omitempty"`

	// Host groups. The write path always uses "groups"; the read path depends on
	// what was selected. Zabbix 7.2 removed selectGroups in favour of
	// selectHostGroups, which returns the membership under "hostgroups" instead.
	// HostsGet folds HostGroups back into GroupIds so callers see one field.
	HostGroups HostGroupIDs `json:"hostgroups,omitempty"`

	// Proxy assignment. Zabbix 7.0 renamed "proxy_hostid" to "proxyid" and made
	// the link explicit via "monitored_by". ProxyID is the version-independent
	// value callers read and write; the wire fields either side of it are
	// populated by prepHosts (write) and HostsGet (read).
	ProxyID      string          `json:"-"`
	ProxyHostID  string          `json:"proxy_hostid,omitempty"` // < 7.0
	ProxyIDField string          `json:"proxyid,omitempty"`      // >= 7.0
	ProxyGroupID string          `json:"proxy_groupid,omitempty"`
	MonitoredBy  MonitoredByType `json:"monitored_by,omitempty"` // >= 7.0
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

	// fix up host details if present
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

		// host groups: >= 7.2 answers selectHostGroups under "hostgroups",
		// everything below answers selectGroups under "groups"
		if len(res[i].GroupIds) == 0 && len(h.HostGroups) > 0 {
			res[i].GroupIds = h.HostGroups
		}

		// proxy: >= 7.0 reports "proxyid", below that "proxy_hostid"
		if api.Config.Version >= V70 {
			res[i].ProxyID = h.ProxyIDField
		} else {
			res[i].ProxyID = h.ProxyHostID
		}
		if res[i].ProxyID == "" {
			res[i].ProxyID = "0"
		}

		res[i].InventoryMode = api.getInvModeFromRaw(&h.RawInventoryMode)

		// tags
		if h.RawTags == nil {
			res[i].Tags = Tags{}
		} else {
			var tags Tags
			if err := json.Unmarshal(h.RawTags, &tags); err != nil {
				api.printf("got error during unmarshal %s", err)
				panic(err)
			}
			res[i].Tags = tags
		}

		// fix up host inventory if present
		if len(h.RawInventory) != 0 {
			// if its an empty array
			asStr := string(h.RawInventory)
			if asStr == "[]" || asStr == "{}" {
				continue
			}

			// lets unbox
			var inv Inventory
			if err := json.Unmarshal(h.RawInventory, &inv); err != nil {
				api.printf("got error during unmarshal %s", err)
				panic(err)
			}

			res[i].Inventory = inv
		}

	}

	return
}

func (api *API) getInvModeFromRaw(raw *json.RawMessage) InventoryMode {
	if raw == nil {
		return InventoryDisabled
	}
	asStr := string(*raw)

	switch asStr {
	case "-1", "\"-1\"":
		return InventoryDisabled
	case "0", "\"0\"":
		return InventoryManual
	case "1", "\"1\"":
		return InventoryAutomatic
	}

	return InventoryDisabled
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
func (api *API) prepHosts(hosts Hosts) {
	for i := 0; i < len(hosts); i++ {
		h := hosts[i]

		// Never send the read-only host group mirror back.
		hosts[i].HostGroups = nil

		// Proxy link. Zabbix 7.0 replaced "proxy_hostid" with "proxyid" and
		// requires "monitored_by" to say who does the monitoring; sending
		// proxyid while monitored_by is 0 (server) is rejected.
		hosts[i].ProxyHostID = ""
		hosts[i].ProxyIDField = ""
		hosts[i].MonitoredBy = ""
		hasProxy := h.ProxyID != "" && h.ProxyID != "0"
		if api.Config.Version >= V70 {
			switch {
			case h.ProxyGroupID != "" && h.ProxyGroupID != "0":
				hosts[i].MonitoredBy = MonitoredByProxyGroup
			case hasProxy:
				hosts[i].MonitoredBy = MonitoredByProxy
				hosts[i].ProxyIDField = h.ProxyID
			default:
				hosts[i].MonitoredBy = MonitoredByServer
				hosts[i].ProxyGroupID = ""
			}
		} else {
			// Before 7.0 there is no monitored_by: proxy_hostid = 0 is how a
			// host is handed back to the server, so it must always be sent.
			hosts[i].ProxyGroupID = ""
			hosts[i].ProxyHostID = h.ProxyID
			if hosts[i].ProxyHostID == "" {
				hosts[i].ProxyHostID = "0"
			}
		}
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
		if h.Tags != nil {
			asB, _ := json.Marshal(h.Tags)
			hosts[i].RawTags = json.RawMessage(asB)
		}
		invMode, _ := json.Marshal(h.InventoryMode)

		hosts[i].RawInventoryMode = json.RawMessage(invMode)
	}
}

// HostsCreate Wrapper for host.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/host/create
func (api *API) HostsCreate(hosts Hosts) (err error) {
	api.prepHosts(hosts)
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
	api.prepHosts(hosts)
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
	var response Response
	response, err = api.CallWithError("host.delete", ids)

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
