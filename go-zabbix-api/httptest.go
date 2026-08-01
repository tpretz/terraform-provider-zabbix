package zabbix

// HTTPTestHeader represents a name/value pair for headers or variables
type HTTPTestHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTPTestHeaders is an array of HTTPTestHeader
type HTTPTestHeaders []HTTPTestHeader

// HTTPTestStep represents a step in a web scenario
type HTTPTestStep struct {
	HTTPStepID      string          `json:"httpstepid,omitempty"`
	Name            string          `json:"name"`
	URL             string          `json:"url"`
	No              string          `json:"no"`
	Timeout         string          `json:"timeout,omitempty"`
	Posts           string          `json:"posts,omitempty"`
	Required        string          `json:"required,omitempty"`
	StatusCodes     string          `json:"status_codes,omitempty"`
	FollowRedirects string          `json:"follow_redirects,omitempty"`
	RetrieveMode    string          `json:"retrieve_mode,omitempty"`
	Headers         HTTPTestHeaders `json:"headers,omitempty"`
	Variables       HTTPTestHeaders `json:"variables,omitempty"`
	QueryFields     HTTPTestHeaders `json:"query_fields,omitempty"`

	// read-only
	HTTPTestID string `json:"httptestid,omitempty"`
	PostType   string `json:"post_type,omitempty"`
}

// HTTPTestSteps is an array of HTTPTestStep
type HTTPTestSteps []HTTPTestStep

// HTTPTest represents a Zabbix web scenario (httptest) object
// https://www.zabbix.com/documentation/7.0/manual/api/reference/httptest/object
type HTTPTest struct {
	HTTPTestID     string          `json:"httptestid,omitempty"`
	Name           string          `json:"name"`
	HostID         string          `json:"hostid,omitempty"`
	Delay          string          `json:"delay,omitempty"`
	Retries        string          `json:"retries,omitempty"`
	Agent          string          `json:"agent,omitempty"`
	HTTPProxy      string          `json:"http_proxy,omitempty"`
	Authentication string          `json:"authentication,omitempty"`
	HTTPUser       string          `json:"http_user,omitempty"`
	HTTPPassword   string          `json:"http_password,omitempty"`
	VerifyPeer     string          `json:"verify_peer,omitempty"`
	VerifyHost     string          `json:"verify_host,omitempty"`
	Status         string          `json:"status,omitempty"`
	Variables      HTTPTestHeaders `json:"variables,omitempty"`
	Headers        HTTPTestHeaders `json:"headers,omitempty"`
	Steps          HTTPTestSteps   `json:"steps,omitempty"`

	// read-only
	TemplateID string `json:"templateid,omitempty"`
}

// HTTPTests is an array of HTTPTest
type HTTPTests []HTTPTest

// HTTPTestsGet Wrapper for httptest.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/httptest/get
func (api *API) HTTPTestsGet(params Params) (res HTTPTests, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("httptest.get", params, &res)
	return
}

// HTTPTestGetByID Gets an httptest by ID only if there is exactly 1 matching httptest.
func (api *API) HTTPTestGetByID(id string) (res *HTTPTest, err error) {
	tests, err := api.HTTPTestsGet(Params{
		"httptestids": id,
		"selectSteps": "extend",
	})
	if err != nil {
		return
	}

	if len(tests) == 1 {
		res = &tests[0]
	} else {
		e := ExpectedOneResult(len(tests))
		err = &e
	}
	return
}

// HTTPTestsCreate Wrapper for httptest.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/httptest/create
func (api *API) HTTPTestsCreate(tests HTTPTests) (err error) {
	response, err := api.CallWithError("httptest.create", tests)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["httptestids"].([]interface{})
	for i, id := range ids {
		tests[i].HTTPTestID = id.(string)
	}
	return
}

// HTTPTestsUpdate Wrapper for httptest.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/httptest/update
func (api *API) HTTPTestsUpdate(tests HTTPTests) (err error) {
	_, err = api.CallWithError("httptest.update", tests)
	return
}

// HTTPTestsDeleteByIds Wrapper for httptest.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/httptest/delete
func (api *API) HTTPTestsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("httptest.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	httptestids := result["httptestids"].([]interface{})
	if len(ids) != len(httptestids) {
		err = &ExpectedMore{len(ids), len(httptestids)}
	}
	return
}
