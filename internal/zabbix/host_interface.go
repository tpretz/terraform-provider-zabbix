package zabbix

import "encoding/json"

type (
	// InterfaceType different interface type
	InterfaceType string
)

const (
	// Differente type of zabbix interface
	// see "type" in https://www.zabbix.com/documentation/3.2/manual/api/reference/hostinterface/object

	// Agent type
	Agent InterfaceType = "1"
	// SNMP type
	SNMP InterfaceType = "2"
	// IPMI type
	IPMI InterfaceType = "3"
	// JMX type
	JMX InterfaceType = "4"
)

// HostInterface represents zabbix host interface type
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostinterface/object
type HostInterface struct {
	InterfaceID string               `json:"interfaceid,omitempty"`
	DNS         string               `json:"dns"`
	IP          string               `json:"ip"`
	Main        string               `json:"main"`
	Port        string               `json:"port"`
	Type        InterfaceType        `json:"type"`
	UseIP       string               `json:"useip"`
	RawDetails  json.RawMessage      `json:"details,omitempty"`
	Details     *HostInterfaceDetail `json:"-"`
}

// HostInterfaces is an array of HostInterface
type HostInterfaces []HostInterface

// HostInterfaceDetail is the SNMP half of a host interface.
//
// Zabbix merges this object key by key into what it already holds: an omitted
// key keeps its stored value, exactly as an omitted scalar property does on
// the object itself. The four v3 credentials therefore carry no omitempty --
// "" is a legitimate value for every one of them (securitylevel
// "noauthnopriv" needs all four empty) and with omitempty the clear was
// dropped, the server kept the old passphrase, and the read put it back into
// state. Verified against 7.4: sending "authpassphrase": "" stores the empty
// value, omitting the key keeps the old one.
//
// Community keeps its omitempty deliberately. It is required and non-empty
// for v1/v2 and ignored for v3, so "" is never a value a working
// configuration wants; the schema default is a macro reference and a
// ValidateFunc forbids the empty string.
//
// The remaining fields are enumerations the provider renders as "0"/"1"/"2".
// omitempty only ever drops "", so it cannot fire on those -- but note how
// close that is: had the codes been ints, "noauthnopriv" (0) could never have
// been selected.
type HostInterfaceDetail struct {
	Version        string `json:"version,omitempty"`
	Bulk           string `json:"bulk,omitempty"`
	Community      string `json:"community,omitempty"`
	SecurityName   string `json:"securityname"`
	SecurityLevel  string `json:"securitylevel,omitempty"`
	AuthPassphrase string `json:"authpassphrase"`
	PrivPassphrase string `json:"privpassphrase"`
	AuthProtocol   string `json:"authprotocol,omitempty"`
	PrivProtocol   string `json:"privprotocol,omitempty"`
	ContextName    string `json:"contextname"`
}

type HostInterfaceDetails []HostInterfaceDetail
