package zabbix

// Proxy represent Zabbix proxy object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/proxy/object
type Proxy struct {
	ProxyID string `json:"proxyid,omitempty"`
	Host    string `json:"host"`
	// add rest later
}

// Proxies is an array of Proxy
type Proxies []Proxy

// ProxiesGet Wrapper for proxy.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/proxy/get
func (api *API) ProxiesGet(params Params) (res Proxies, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("proxy.get", params, &res)
	return
}
