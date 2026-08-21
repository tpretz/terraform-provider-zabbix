package zabbix

import (
	"encoding/json"
	"strings"
)

// redactPlaceholder is what a secret is replaced with in the logs. It is
// deliberately not the empty string: a reader has to be able to tell "the
// field was sent and has been hidden" from "the field was not sent", because
// that distinction is exactly what most of this provider's bugs turned on.
const redactPlaceholder = "***REDACTED***"

// sensitiveKeys are the JSON property names whose values must never reach a
// log. Matching is on the property name, case-insensitively, at any depth.
//
// The list covers what the provider sends today plus the names Zabbix uses for
// credentials the provider does not expose yet (ssh keys, user passwords),
// because the cost of a name being here early is nothing and the cost of it
// being here late is a disclosed secret.
var sensitiveKeys = map[string]bool{
	// the session token, as the JSON-RPC body property below Zabbix 6.4
	"auth": true,
	// provider credentials, and user.* methods
	"password":       true,
	"passwd":         true,
	"current_passwd": true,
	"token":          true,
	"sessionid":      true,
	// host and proxy pre-shared keys
	"tls_psk":          true,
	"tls_psk_identity": true,
	// host IPMI
	"ipmi_password": true,
	// SNMPv3, on the host interface details object
	"authpassphrase":        true,
	"privpassphrase":        true,
	"snmpv3_authpassphrase": true,
	"snmpv3_privpassphrase": true,
	"snmp_community":        true,
	// ssh/telnet item credentials, for when those backends land
	"privatekey": true,
	"publickey":  true,
}

// redactBody returns a loggable rendering of a JSON-RPC body with every
// sensitive value replaced.
//
// It fails closed. If the body cannot be parsed it returns a placeholder
// rather than the original bytes, because the one case where parsing fails is
// the one case where nobody has checked what the bytes contain.
func redactBody(b []byte) string {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return "<unloggable: body is not JSON>"
	}
	out, err := json.Marshal(redactValue(v, false))
	if err != nil {
		return "<unloggable: redacted body would not marshal>"
	}
	return string(out)
}

// redactResponseBody is redactBody plus the one thing a key name cannot
// express: user.login answers with the session token as its bare "result", so
// for that method the result itself is the secret.
func redactResponseBody(method string, b []byte) string {
	if !strings.EqualFold(method, "user.login") {
		return redactBody(b)
	}
	var v map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return "<unloggable: body is not JSON>"
	}
	if _, ok := v["result"]; ok {
		v["result"] = redactPlaceholder
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "<unloggable: redacted body would not marshal>"
	}
	return string(out)
}

// redactValue walks a decoded JSON document. inSensitive carries down through
// nested structures so that, say, a whole "password" object is hidden rather
// than only its scalar leaves.
func redactValue(v interface{}, inSensitive bool) interface{} {
	if inSensitive {
		return redactPlaceholder
	}

	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			out[k] = redactValue(val, sensitiveKeys[strings.ToLower(k)])
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = redactValue(val, false)
		}
		return out
	default:
		return v
	}
}

// RedactedJSON renders any value as JSON with sensitive properties hidden, for
// callers outside this package that want to log a struct.
//
// The provider logs whole objects at [DEBUG] — `log.Trace("created host: %+v",
// host)` and friends — and those structs carry TLSPSK, IPMIPassword and the
// SNMPv3 passphrases as plain strings, so %+v prints them. Going through the
// struct's own JSON tags means this shares one list of sensitive names with the
// request logging rather than keeping a second one in step.
//
// Like redactBody it fails closed: a value that will not marshal returns a
// placeholder, never the original.
func RedactedJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "<unloggable: value would not marshal>"
	}
	return redactBody(b)
}
