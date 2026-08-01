package zabbix

// RegexpExpressionType represents the type of a regexp expression
type RegexpExpressionType string

const (
	RegexpExpressionCharIncluded    RegexpExpressionType = "0"
	RegexpExpressionCharNotIncluded RegexpExpressionType = "1"
	RegexpExpressionRegexp          RegexpExpressionType = "2"
	RegexpExpressionNotRegexp       RegexpExpressionType = "3"
	RegexpExpressionAnyIncluded     RegexpExpressionType = "4"
)

// RegexpExpression represents a single expression entry in a regexp
type RegexpExpression struct {
	Expression     string               `json:"expression"`
	ExpressionType RegexpExpressionType `json:"expression_type"`
	ExpDelimiter   string               `json:"exp_delimiter"`
	CaseSensitive  string               `json:"case_sensitive"`
}

// RegexpExpressions is an array of RegexpExpression
type RegexpExpressions []RegexpExpression

// Regexp represents a Zabbix global regular expression object
// https://www.zabbix.com/documentation/7.0/manual/api/reference/regexp/object
type Regexp struct {
	RegexpID    string            `json:"regexpid,omitempty"`
	Name        string            `json:"name"`
	TestString  string            `json:"test_string"`
	Expressions RegexpExpressions `json:"expressions,omitempty"`
}

// Regexps is an array of Regexp
type Regexps []Regexp

// RegexpsGet Wrapper for regexp.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/regexp/get
func (api *API) RegexpsGet(params Params) (res Regexps, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("regexp.get", params, &res)
	return
}

// RegexpGetByID Gets a regexp by ID only if there is exactly 1 matching.
func (api *API) RegexpGetByID(id string) (res *Regexp, err error) {
	regexps, err := api.RegexpsGet(Params{"regexpids": id, "selectExpressions": "extend"})
	if err != nil {
		return
	}

	if len(regexps) == 1 {
		res = &regexps[0]
	} else {
		e := ExpectedOneResult(len(regexps))
		err = &e
	}
	return
}

// RegexpsCreate Wrapper for regexp.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/regexp/create
func (api *API) RegexpsCreate(regexps Regexps) (err error) {
	response, err := api.CallWithError("regexp.create", regexps)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["regexpids"].([]interface{})
	for i, id := range ids {
		regexps[i].RegexpID = id.(string)
	}
	return
}

// RegexpsUpdate Wrapper for regexp.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/regexp/update
func (api *API) RegexpsUpdate(regexps Regexps) (err error) {
	_, err = api.CallWithError("regexp.update", regexps)
	return
}

// RegexpsDeleteByIds Wrapper for regexp.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/regexp/delete
func (api *API) RegexpsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("regexp.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	regexpids := result["regexpids"].([]interface{})
	if len(ids) != len(regexpids) {
		err = &ExpectedMore{len(ids), len(regexpids)}
	}
	return
}
