package zabbix

// Proxy represent Zabbix proxy object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/proxy/object
type Proxy struct {
	ProxyID string `json:"proxyid,omitempty"`
	// Zabbix 7.0 renamed the proxy's technical name from "host" to "name" and
	// dropped the separate proxy interface object. ProxiesGet folds Name back
	// into Host so callers see one field.
	Host string `json:"host,omitempty"`
	Name string `json:"name,omitempty"`
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
	for i := range res {
		if res[i].Host == "" {
			res[i].Host = res[i].Name
		}
	}
	return
}
