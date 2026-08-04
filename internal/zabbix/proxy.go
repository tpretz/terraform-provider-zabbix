package zabbix

import (
	"encoding/json"
	"net"
	"strconv"
)

type (
	// ProxyMode is a proxy's operating mode, in the version-neutral encoding
	// this package uses. The values are Zabbix 7.0's "operating_mode" codes;
	// before 7.0 the same concept is the "status" property with the entirely
	// different codes 5 (active) and 6 (passive), and proxyParams translates.
	ProxyMode int

	// TLSMode is a Zabbix encryption mode, as used by the proxy properties
	// "tls_connect" (how the server connects to a passive proxy) and
	// "tls_accept" (what the server accepts from an active proxy). The values
	// are the API's own numeric codes and have not changed across any
	// supported version.
	TLSMode string
)

const (
	// ProxyModeActive the proxy connects to the server ("status" 5 before 7.0)
	ProxyModeActive ProxyMode = 0
	// ProxyModePassive the server connects to the proxy ("status" 6 before 7.0)
	ProxyModePassive ProxyMode = 1
)

const (
	// TLSUnencrypted no encryption (the default)
	TLSUnencrypted TLSMode = "1"
	// TLSPSKMode pre-shared key
	TLSPSKMode TLSMode = "2"
	// TLSCertificate certificate
	TLSCertificate TLSMode = "4"
)

// Defaults Zabbix itself applies to a proxy's connection endpoint. 7.0 returns
// these verbatim for an active proxy (where they are meaningless) and rejects
// any other value, so the provider treats them as the "unset" representation
// on every version.
const (
	// ProxyDefaultAddress default address of a passive proxy
	ProxyDefaultAddress = "127.0.0.1"
	// ProxyDefaultPort default port of a passive proxy
	ProxyDefaultPort = "10051"
)

// Proxy is the version-neutral representation of a Zabbix proxy.
//
// Zabbix 7.0 rewrote this object wholesale — every mode-dependent property was
// renamed, and the separate proxy interface was folded into the proxy itself:
//
//	before 7.0            7.0 and later
//	host                  name
//	status (5/6)          operating_mode (0/1)
//	interface{ip,dns,port} address + port
//	proxy_address         allowed_addresses
//
// Callers use the fields below on every supported version; ProxiesGet and
// proxyParams do the translation in one place.
//
// https://www.zabbix.com/documentation/current/en/manual/api/reference/proxy/object
type Proxy struct {
	ProxyID string
	// Name is the proxy's technical name ("host" before 7.0, "name" from 7.0).
	Name string
	// Mode is active or passive.
	Mode ProxyMode
	// Address and Port are where the server connects to reach a PASSIVE proxy.
	// Before 7.0 they live in a nested interface object, from 7.0 they are
	// plain properties. Both are reported empty for an active proxy, which has
	// no endpoint of its own on any version.
	Address string
	Port    string
	// AllowedAddresses restricts the addresses an ACTIVE proxy may connect
	// from ("proxy_address" before 7.0, "allowed_addresses" from 7.0). Empty
	// for a passive proxy.
	AllowedAddresses string
	Description      string
	// TLSConnect applies to passive proxies and TLSAccept to active ones.
	// Zabbix rejects a non-default value on the mode it does not apply to.
	TLSConnect TLSMode
	TLSAccept  TLSMode
	TLSIssuer  string
	TLSSubject string
	// TLSPSKIdentity and TLSPSK are write-only: every supported version
	// accepts them on create/update and none of them ever returns them from
	// proxy.get, so ProxiesGet always leaves them empty.
	TLSPSKIdentity string
	TLSPSK         string
}

// Proxies is an array of Proxy
type Proxies []Proxy

// proxyResponse is the wire form of a proxy as returned by proxy.get. It is
// the union of the pre-7.0 and 7.0+ property sets; which half is populated
// depends on the server, so nothing here is used without a version check.
type proxyResponse struct {
	ProxyID string `json:"proxyid"`

	// < 7.0
	Host         string `json:"host"`
	Status       string `json:"status"`
	ProxyAddress string `json:"proxy_address"`
	// Interface is an object for a passive proxy and an empty ARRAY for a
	// proxy that has none, so it cannot be decoded straight into a struct.
	Interface json.RawMessage `json:"interface"`

	// >= 7.0
	Name             string `json:"name"`
	OperatingMode    string `json:"operating_mode"`
	Address          string `json:"address"`
	Port             string `json:"port"`
	AllowedAddresses string `json:"allowed_addresses"`

	// common
	Description string `json:"description"`
	TLSConnect  string `json:"tls_connect"`
	TLSAccept   string `json:"tls_accept"`
	TLSIssuer   string `json:"tls_issuer"`
	TLSSubject  string `json:"tls_subject"`
}

// proxyInterface is the pre-7.0 nested proxy interface object.
type proxyInterface struct {
	UseIP string `json:"useip"`
	IP    string `json:"ip"`
	DNS   string `json:"dns"`
	Port  string `json:"port"`
}

// ProxyNameProperty returns the API property holding a proxy's technical name
// on this server: "host" before 7.0, "name" from 7.0. Callers need it to build
// filter/search parameters, which are passed through to the API untouched.
func (api *API) ProxyNameProperty() string {
	if api.Config.Version >= V70 {
		return "name"
	}
	return "host"
}

// ProxiesGet Wrapper for proxy.get
// https://www.zabbix.com/documentation/current/en/manual/api/reference/proxy/get
func (api *API) ProxiesGet(params Params) (res Proxies, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	// Before 7.0 a passive proxy's address lives in a separate interface
	// object that has to be asked for explicitly. From 7.0 there is no such
	// object and "selectInterface" is rejected as an unknown parameter.
	if api.Config.Version < V70 {
		if _, present := params["selectInterface"]; !present {
			params["selectInterface"] = "extend"
		}
	}

	var wire []proxyResponse
	if err = api.CallWithErrorParse("proxy.get", params, &wire); err != nil {
		return
	}

	res = make(Proxies, len(wire))
	for i, w := range wire {
		res[i] = api.proxyFromResponse(w)
	}
	return
}

// ProxyGetByID Gets proxy by Id only if there is exactly 1 matching proxy.
func (api *API) ProxyGetByID(id string) (res *Proxy, err error) {
	proxies, err := api.ProxiesGet(Params{"proxyids": id})
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

// proxyFromResponse folds one version's property set into the neutral Proxy.
func (api *API) proxyFromResponse(w proxyResponse) (p Proxy) {
	p = Proxy{
		ProxyID:     w.ProxyID,
		Description: w.Description,
		TLSConnect:  TLSMode(w.TLSConnect),
		TLSAccept:   TLSMode(w.TLSAccept),
		TLSIssuer:   w.TLSIssuer,
		TLSSubject:  w.TLSSubject,
	}

	if api.Config.Version >= V70 {
		p.Name = w.Name
		if w.OperatingMode == strconv.Itoa(int(ProxyModePassive)) {
			p.Mode = ProxyModePassive
			// 7.0 reports address/port for an active proxy too, always at
			// their defaults. Reporting them only for a passive proxy keeps
			// the field meaning the same as it does before 7.0, where an
			// active proxy simply has no interface.
			p.Address = w.Address
			p.Port = w.Port
		}
		p.AllowedAddresses = w.AllowedAddresses
		return
	}

	p.Name = w.Host
	if w.Status == "6" {
		p.Mode = ProxyModePassive
	}
	p.AllowedAddresses = w.ProxyAddress

	if p.Mode == ProxyModePassive && len(w.Interface) > 0 {
		var iface proxyInterface
		// an interfaceless proxy reports "interface": [], which is not an
		// error worth surfacing -- it just means there is no address.
		if json.Unmarshal(w.Interface, &iface) == nil {
			if iface.UseIP == "1" {
				p.Address = iface.IP
			} else {
				p.Address = iface.DNS
			}
			p.Port = iface.Port
		}
	}
	return
}

// proxyParams renders a Proxy into the create/update request object for this
// server's API version.
//
// Zabbix validates a proxy strictly by operating mode: for an ACTIVE proxy it
// rejects any "address"/"port" other than the defaults and any "tls_connect"
// other than 1, and for a PASSIVE proxy it rejects a non-empty
// "allowed_addresses" and any "tls_accept" other than 1. The mode-specific
// properties are therefore only sent for the mode they belong to — except the
// TLS pair, which is always sent so that a configuration asking for encryption
// on the wrong side is reported by the server rather than silently dropped.
func (api *API) proxyParams(p Proxy) Params {
	tlsConnect := p.TLSConnect
	if tlsConnect == "" {
		tlsConnect = TLSUnencrypted
	}
	tlsAccept := p.TLSAccept
	if tlsAccept == "" {
		tlsAccept = TLSUnencrypted
	}

	// Every property below is sent unconditionally, empty string included:
	// omitting one on update means "leave as is", which would make a cleared
	// field in Terraform unclearable in Zabbix.
	out := Params{
		"description":      p.Description,
		"tls_connect":      string(tlsConnect),
		"tls_accept":       string(tlsAccept),
		"tls_issuer":       p.TLSIssuer,
		"tls_subject":      p.TLSSubject,
		"tls_psk_identity": p.TLSPSKIdentity,
		"tls_psk":          p.TLSPSK,
	}

	if api.Config.Version >= V70 {
		out["name"] = p.Name
		out["operating_mode"] = strconv.Itoa(int(p.Mode))
		if p.Mode == ProxyModePassive {
			out["address"] = proxyAddressOrDefault(p.Address)
			out["port"] = proxyPortOrDefault(p.Port)
		} else {
			out["allowed_addresses"] = p.AllowedAddresses
		}
		return out
	}

	out["host"] = p.Name
	if p.Mode == ProxyModePassive {
		out["status"] = "6"
		out["interface"] = proxyInterfaceParams(p)
	} else {
		out["status"] = "5"
		out["proxy_address"] = p.AllowedAddresses
		// An active proxy must have no interface. Omitting the property on
		// update is enough: Zabbix drops the existing interface when the
		// proxy stops being passive.
	}
	return out
}

// proxyInterfaceParams builds the pre-7.0 nested interface object for a
// passive proxy. 7.0's single "address" property covers both an IP and a DNS
// name, so the ip/dns/useip split is derived from the value's own shape.
func proxyInterfaceParams(p Proxy) Params {
	address := proxyAddressOrDefault(p.Address)
	iface := Params{
		"useip": "0",
		"ip":    "",
		"dns":   address,
		"port":  proxyPortOrDefault(p.Port),
	}
	if net.ParseIP(address) != nil {
		iface["useip"] = "1"
		iface["ip"] = address
		iface["dns"] = ""
	}
	return iface
}

func proxyAddressOrDefault(address string) string {
	if address == "" {
		return ProxyDefaultAddress
	}
	return address
}

func proxyPortOrDefault(port string) string {
	if port == "" {
		return ProxyDefaultPort
	}
	return port
}

// ProxiesCreate Wrapper for proxy.create
// https://www.zabbix.com/documentation/current/en/manual/api/reference/proxy/create
func (api *API) ProxiesCreate(proxies Proxies) (err error) {
	params := make([]Params, len(proxies))
	for i, p := range proxies {
		params[i] = api.proxyParams(p)
	}

	response, err := api.CallWithError("proxy.create", params)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	proxyids := result["proxyids"].([]interface{})
	for i, id := range proxyids {
		proxies[i].ProxyID = id.(string)
	}
	return
}

// ProxiesUpdate Wrapper for proxy.update
// https://www.zabbix.com/documentation/current/en/manual/api/reference/proxy/update
func (api *API) ProxiesUpdate(proxies Proxies) (err error) {
	params := make([]Params, len(proxies))
	for i, p := range proxies {
		params[i] = api.proxyParams(p)
		params[i]["proxyid"] = p.ProxyID
	}

	_, err = api.CallWithError("proxy.update", params)
	return
}

// ProxiesDelete Wrapper for proxy.delete
// Cleans ProxyID in all proxies elements if the call succeeds.
// https://www.zabbix.com/documentation/current/en/manual/api/reference/proxy/delete
func (api *API) ProxiesDelete(proxies Proxies) (err error) {
	ids := make([]string, len(proxies))
	for i, proxy := range proxies {
		ids[i] = proxy.ProxyID
	}

	err = api.ProxiesDeleteByIds(ids)
	if err == nil {
		for i := range proxies {
			proxies[i].ProxyID = ""
		}
	}
	return
}

// ProxiesDeleteByIds Wrapper for proxy.delete
// https://www.zabbix.com/documentation/current/en/manual/api/reference/proxy/delete
func (api *API) ProxiesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("proxy.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	proxyids := result["proxyids"].([]interface{})
	if len(ids) != len(proxyids) {
		err = &ExpectedMore{len(ids), len(proxyids)}
	}
	return
}
