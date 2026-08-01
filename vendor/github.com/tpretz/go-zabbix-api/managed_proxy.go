package zabbix

// ManagedProxy represents a Zabbix 7.0 proxy object for create/update operations.
// The field names changed in 7.0: 'host' -> 'name', 'status' -> 'operating_mode'.
// tls_psk and tls_psk_identity are write-only (not returned by proxy.get).
type ManagedProxy struct {
	ProxyID          string `json:"proxyid,omitempty"`
	Name             string `json:"name"`
	OperatingMode    string `json:"operating_mode,omitempty"`
	Description      string `json:"description,omitempty"`
	AllowedAddresses string `json:"allowed_addresses,omitempty"`
	Address          string `json:"address,omitempty"`
	Port             string `json:"port,omitempty"`
	TLSConnect       string `json:"tls_connect,omitempty"`
	TLSAccept        string `json:"tls_accept,omitempty"`
	TLSIssuer        string `json:"tls_issuer,omitempty"`
	TLSSubject       string `json:"tls_subject,omitempty"`
	TLSPSKIdentity   string `json:"tls_psk_identity,omitempty"`
	TLSPSK           string `json:"tls_psk,omitempty"`
	CustomTimeouts   string `json:"custom_timeouts,omitempty"`

	TimeoutZabbixAgent   string `json:"timeout_zabbix_agent,omitempty"`
	TimeoutSimpleCheck   string `json:"timeout_simple_check,omitempty"`
	TimeoutSNMPAgent     string `json:"timeout_snmp_agent,omitempty"`
	TimeoutExternalCheck string `json:"timeout_external_check,omitempty"`
	TimeoutDBMonitor     string `json:"timeout_db_monitor,omitempty"`
	TimeoutHTTPAgent     string `json:"timeout_http_agent,omitempty"`
	TimeoutSSHAgent      string `json:"timeout_ssh_agent,omitempty"`
	TimeoutTelnetAgent   string `json:"timeout_telnet_agent,omitempty"`
	TimeoutScript        string `json:"timeout_script,omitempty"`
	TimeoutBrowser       string `json:"timeout_browser,omitempty"`

	ProxyGroupID string `json:"proxy_groupid,omitempty"`
	LocalAddress string `json:"local_address,omitempty"`
	LocalPort    string `json:"local_port,omitempty"`

	// read-only fields
	LastAccess    string `json:"lastaccess,omitempty"`
	Version       string `json:"version,omitempty"`
	Compatibility string `json:"compatibility,omitempty"`
	State         string `json:"state,omitempty"`
}

// ManagedProxies is an array of ManagedProxy
type ManagedProxies []ManagedProxy

// ManagedProxiesGet Wrapper for proxy.get returning ManagedProxy structs (7.0 fields)
func (api *API) ManagedProxiesGet(params Params) (res ManagedProxies, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("proxy.get", params, &res)
	return
}

// ManagedProxyGetByID Gets a proxy by ID only if there is exactly 1 matching result.
func (api *API) ManagedProxyGetByID(id string) (res *ManagedProxy, err error) {
	proxies, err := api.ManagedProxiesGet(Params{"proxyids": id})
	if err != nil {
		return
	}
	if len(proxies) == 1 {
		res = &proxies[0]
	} else {
		e := ExpectedOneResult(len(proxies))
		err = &e
	}
	return
}

// ManagedProxiesCreate Wrapper for proxy.create
func (api *API) ManagedProxiesCreate(proxies ManagedProxies) (err error) {
	response, err := api.CallWithError("proxy.create", proxies)
	if err != nil {
		return
	}
	result := response.Result.(map[string]interface{})
	ids := result["proxyids"].([]interface{})
	for i, id := range ids {
		proxies[i].ProxyID = id.(string)
	}
	return
}

// ManagedProxiesUpdate Wrapper for proxy.update
func (api *API) ManagedProxiesUpdate(proxies ManagedProxies) (err error) {
	_, err = api.CallWithError("proxy.update", proxies)
	return
}

// ManagedProxiesDeleteByIds Wrapper for proxy.delete
func (api *API) ManagedProxiesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("proxy.delete", ids)
	if err != nil {
		return
	}
	result := response.Result.(map[string]interface{})
	deletedIds := result["proxyids"].([]interface{})
	if len(ids) != len(deletedIds) {
		err = &ExpectedMore{len(ids), len(deletedIds)}
	}
	return
}
