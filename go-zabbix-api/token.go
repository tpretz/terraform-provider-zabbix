package zabbix

// TokenStatus is the enabled/disabled state of an API token
type TokenStatus string

const (
	// TokenEnabled token may be used to authenticate
	TokenEnabled TokenStatus = "0"
	// TokenDisabled token is rejected
	TokenDisabled TokenStatus = "1"
)

// Token represents a Zabbix API token.
//
// The token secret itself is only ever returned by token.generate, and only
// once; token.get never exposes it.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/token/object
type Token struct {
	TokenID     string      `json:"tokenid,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	UserID      string      `json:"userid,omitempty"`
	Status      TokenStatus `json:"status,omitempty"`
	// ExpiresAt is a unix timestamp, "0" means the token never expires
	ExpiresAt string `json:"expires_at,omitempty"`

	// read only
	LastAccess    string `json:"lastaccess,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	CreatorUserID string `json:"creator_userid,omitempty"`
}

// Tokens is an array of Token
type Tokens []Token

// TokenSecret is the result of token.generate
type TokenSecret struct {
	TokenID string `json:"tokenid"`
	Token   string `json:"token"`
}

// TokenSecrets is an array of TokenSecret
type TokenSecrets []TokenSecret

// TokensGet Wrapper for token.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/token/get
func (api *API) TokensGet(params Params) (res Tokens, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("token.get", params, &res)
	return
}

// TokenGetByID Gets a token by ID only if there is exactly 1 matching token.
func (api *API) TokenGetByID(id string) (res *Token, err error) {
	tokens, err := api.TokensGet(Params{"tokenids": id})
	if err != nil {
		return
	}

	if len(tokens) == 1 {
		res = &tokens[0]
	} else {
		e := ExpectedOneResult(len(tokens))
		err = &e
	}
	return
}

// TokensCreate Wrapper for token.create
// The returned Tokens have their TokenID populated. Use TokensGenerate to
// obtain the secret.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/token/create
func (api *API) TokensCreate(tokens Tokens) (res Tokens, err error) {
	response, err := api.CallWithError("token.create", tokens)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	tokenids := result["tokenids"].([]interface{})
	res = make(Tokens, len(tokens))
	copy(res, tokens)
	for i, id := range tokenids {
		res[i].TokenID = id.(string)
	}
	return
}

// TokensUpdate Wrapper for token.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/token/update
func (api *API) TokensUpdate(tokens Tokens) (err error) {
	_, err = api.CallWithError("token.update", tokens)
	return
}

// TokensGenerate Wrapper for token.generate
// Generates (or regenerates) the secret for the given token IDs. This is the
// only way to obtain the secret, and it invalidates any previous secret.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/token/generate
func (api *API) TokensGenerate(ids []string) (res TokenSecrets, err error) {
	err = api.CallWithErrorParse("token.generate", ids, &res)
	return
}

// TokensDeleteByIds Wrapper for token.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/token/delete
func (api *API) TokensDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("token.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	tokenids := result["tokenids"].([]interface{})
	if len(ids) != len(tokenids) {
		err = &ExpectedMore{len(ids), len(tokenids)}
	}
	return
}
