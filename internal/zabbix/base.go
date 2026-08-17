package zabbix

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type (
	// Params Zabbix request param
	Params map[string]interface{}
)

// Zabbix API version gates. Config.Version is encoded by parseVersionString as
// major*10000 + minor*100 + patch — 6.0.13 is 60013 and 7.4.13 is 70413 — so a
// minor release is 100, not 1000, apart. (PLAN.md and API-COVERAGE.md quote
// these gates as 62000/64000/72000/74000; under this encoding those are
// versions 6.20/6.40/7.20/7.40 and would never match. The names below are the
// documented ones, the values are the ones that actually compare correctly.)
// Use these instead of bare integers so version-conditional behaviour stays
// greppable.
const (
	// V62 6.2: template groups split out from host groups
	V62 = 60200
	// V64 6.4: "Authorization: Bearer" header accepted; template vendor fields
	V64 = 60400
	// V70 7.0: proxy model rewrite (proxyid/monitored_by), HTTP header arrays,
	// item.create/discoveryrule.create reject unknown object properties,
	// preprocessing "check for not supported value" requires params
	V70 = 70000
	// V72 7.2: "auth" request property removed; selectGroups replaced by
	// selectHostGroups / selectTemplateGroups
	V72 = 70200
	// V74 7.4: LLD rule prototypes
	V74 = 70400
)

type request struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Auth    string      `json:"auth,omitempty"`
	ID      int32       `json:"id"`
}

// Response format of zabbix api
type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Error   *Error      `json:"error"`
	Result  interface{} `json:"result"`
	ID      int32       `json:"id"`
}

// RawResponse format of zabbix api
type RawResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Error   *Error          `json:"error"`
	Result  json.RawMessage `json:"result"`
	ID      int32           `json:"id"`
}

// Error contains error data and code
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d (%s): %s", e.Code, e.Message, e.Data)
}

// ExpectedOneResult use to generate error when you expect one result
type ExpectedOneResult int

func (e *ExpectedOneResult) Error() string {
	return fmt.Sprintf("Expected exactly one result, got %d.", *e)
}

// ExpectedMore use to generate error when you expect more element
type ExpectedMore struct {
	Expected int
	Got      int
}

func (e *ExpectedMore) Error() string {
	return fmt.Sprintf("Expected %d, got %d.", e.Expected, e.Got)
}

// API use to store connection information
type API struct {
	Auth      string      // auth token, filled by Login()
	Logger    *log.Logger // request/response logger, nil by default
	UserAgent string
	url       string
	c         http.Client
	id        int32
	ex        sync.Mutex
	Config    Config
}

type Config struct {
	Url         string
	TlsNoVerify bool
	Log         *log.Logger
	Serialize   bool
	Version     int
}

func parseVersionString(vstr string) (version int64, err error) {
	parts := strings.Split(vstr, ".")

	version, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}
	version = version * 10000

	// do we have a minor version
	if len(parts) > 1 {
		var no int64
		no, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return
		}
		version += no * 100
	}

	// do we have a patch version
	if len(parts) > 2 {
		var no int64
		no, err = strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return
		}
		version += no
	}
	return
}

// NewAPI Creates new API access object.
// Typical URL is http://host/api_jsonrpc.php or http://host/zabbix/api_jsonrpc.php.
// It also may contain HTTP basic auth username and password like
// http://username:password@host/api_jsonrpc.php.
func NewAPI(c Config) (api *API, err error) {
	api = &API{
		url:       c.Url,
		c:         http.Client{},
		UserAgent: "github.com/tpretz/terraform-provider-zabbix",
		Logger:    c.Log,
		Config:    c,
	}

	if c.TlsNoVerify {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		api.c = http.Client{
			Transport: tr,
		}
		api.printf("TLS running in insecure mode, do not use this configuration in production")
	}

	var rawVersion string
	rawVersion, err = api.Version()
	if err != nil {
		return
	}
	var version int64
	version, err = parseVersionString(rawVersion)
	if err != nil {
		return
	}
	api.Config.Version = int(version)

	return
}

// SetClient Allows one to use specific http.Client, for example with InsecureSkipVerify transport.
func (api *API) SetClient(c *http.Client) {
	api.c = *c
}

func (api *API) printf(format string, v ...interface{}) {
	if api.Logger != nil {
		api.Logger.Printf(format, v...)
	}
}

// isReadMethod reports whether a JSON-RPC method only reads.
//
// Every Zabbix read method is named "<object>.get", with apiinfo.version the
// one exception. Everything else — create, update, delete, and the mass*
// variants — mutates, and it is the mutating calls that Serialize exists to
// hold apart. Matching on the suffix rather than a list means a method added
// later is treated as a write, which is the safe way to be wrong.
func isReadMethod(method string) bool {
	return strings.HasSuffix(method, ".get") || strings.EqualFold(method, "apiinfo.version")
}

func (api *API) callBytes(method string, params interface{}) (b []byte, err error) {
	id := atomic.AddInt32(&api.id, 1)

	// Zabbix 6.4 started accepting the auth token in an "Authorization: Bearer"
	// header, and 7.2 removed the "auth" JSON-RPC body property entirely — it is
	// now rejected as an unexpected parameter, which fails every call. Send
	// exactly one of the two, never both. Config.Version is 0 until the
	// unauthenticated apiinfo.version probe in NewAPI has run, and api.Auth is
	// empty until Login(), so neither is attached to that probe.
	useBearer := api.Auth != "" && api.Config.Version >= V64

	jsonobj := request{Jsonrpc: "2.0", Method: method, Params: params, ID: id}
	if !useBearer {
		jsonobj.Auth = api.Auth
	}
	b, err = json.Marshal(jsonobj)
	if err != nil {
		return
	}
	api.printf("Request (POST): %s", b)

	req, err := http.NewRequest("POST", api.url, bytes.NewReader(b))
	if err != nil {
		return
	}
	req.ContentLength = int64(len(b))
	req.Header.Add("Content-Type", "application/json-rpc")
	req.Header.Add("User-Agent", api.UserAgent)
	if useBearer {
		req.Header.Add("Authorization", "Bearer "+api.Auth)
	}

	// Serialize holds writes apart. Zabbix's template inheritance does
	// read-modify-write against shared parent objects and is not safe against
	// concurrent callers, and Terraform applies at parallelism 10 by default —
	// the ordinary shape of a configuration, one template with many hosts
	// linking it, drives exactly the racy path. Observed in the wild: a host
	// linked to a template ended up with the template's items and none of its
	// triggers, silently, and only surfaced weeks later when an unrelated
	// dependency tripped over the gap.
	//
	// Reads are left concurrent. A ".get" cannot corrupt anything, and refresh
	// and plan are almost entirely reads, so locking them would cost real time
	// and buy no safety.
	if api.Config.Serialize && !isReadMethod(method) {
		api.ex.Lock()
		defer api.ex.Unlock()
	}

	res, err := api.c.Do(req)
	if err != nil {
		api.printf("Error   : %s", err)
		return
	}
	defer res.Body.Close()

	b, err = ioutil.ReadAll(res.Body)
	api.printf("Response (%d): %s", res.StatusCode, b)
	return
}

// Call Calls specified API method. Uses api.Auth if not empty.
// err is something network or marshaling related. Caller should inspect response.Error to get API error.
func (api *API) Call(method string, params interface{}) (response Response, err error) {
	b, err := api.callBytes(method, params)
	if err == nil {
		err = json.Unmarshal(b, &response)
	}
	return
}

// CallWithError Uses Call() and then sets err to response.Error if former is nil and latter is not.
func (api *API) CallWithError(method string, params interface{}) (response Response, err error) {
	response, err = api.Call(method, params)
	if err == nil && response.Error != nil {
		err = response.Error
	}
	return
}

// CallWithErrorParse Calls specified API method.
// Parse the response of the api in the result variable.
func (api *API) CallWithErrorParse(method string, params interface{}, result interface{}) (err error) {
	var rawResult RawResponse

	response, err := api.callBytes(method, params)
	if err != nil {
		return
	}
	err = json.Unmarshal(response, &rawResult)
	if err != nil {
		return
	}
	if rawResult.Error != nil {
		return rawResult.Error
	}
	err = json.Unmarshal(rawResult.Result, &result)
	return
}

// Login Calls "user.login" API method and fills api.Auth field.
// This method modifies API structure and should not be called concurrently with other methods.
func (api *API) Login(user, password string) (auth string, err error) {
	response, err := api.CallWithError("user.login", map[string]string{"username": user, "password": password})
	if err != nil {
		return
	}

	auth = response.Result.(string)
	api.Auth = auth
	return
}

// Version Calls "APIInfo.version" API method.
// This method temporary modifies API structure and should not be called concurrently with other methods.
func (api *API) Version() (v string, err error) {
	// temporary remove auth for this method to succeed
	// https://www.zabbix.com/documentation/2.2/manual/appendix/api/apiinfo/version
	auth := api.Auth
	api.Auth = ""
	response, err := api.CallWithError("APIInfo.version", Params{})
	api.Auth = auth

	// despite what documentation says, Zabbix 2.2 requires auth, so we try again
	if e, ok := err.(*Error); ok && e.Code == -32602 {
		response, err = api.CallWithError("APIInfo.version", Params{})
	}
	if err != nil {
		return
	}

	v = response.Result.(string)
	return
}
