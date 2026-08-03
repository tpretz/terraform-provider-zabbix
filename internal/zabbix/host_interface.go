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

type HostInterfaceDetail struct {
	Version        string `json:"version,omitempty"`
	Bulk           string `json:"bulk,omitempty"`
	Community      string `json:"community,omitempty"`
	SecurityName   string `json:"securityname,omitempty"`
	SecurityLevel  string `json:"securitylevel,omitempty"`
	AuthPassphrase string `json:"authpassphrase,omitempty"`
	PrivPassphrase string `json:"privpassphrase,omitempty"`
	AuthProtocol   string `json:"authprotocol,omitempty"`
	PrivProtocol   string `json:"privprotocol,omitempty"`
	ContextName    string `json:"contextname,omitempty"`
}

type HostInterfaceDetails []HostInterfaceDetail
