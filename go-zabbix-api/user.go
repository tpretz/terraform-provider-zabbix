package zabbix

// User represents a Zabbix user object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/user/object
type User struct {
	UserID      string      `json:"userid,omitempty"`
	Username    string      `json:"username"`
	Name        string      `json:"name,omitempty"`
	Surname     string      `json:"surname,omitempty"`
	Passwd      string      `json:"passwd,omitempty"`
	RoleID      string      `json:"roleid,omitempty"`
	Autologin   string      `json:"autologin,omitempty"`
	Autologout  string      `json:"autologout,omitempty"`
	Lang        string      `json:"lang,omitempty"`
	Refresh     string      `json:"refresh,omitempty"`
	RowsPerPage string      `json:"rows_per_page,omitempty"`
	Theme       string      `json:"theme,omitempty"`
	Timezone    string      `json:"timezone,omitempty"`
	URL         string      `json:"url,omitempty"`
	UserGroups  []UserGrpID `json:"usrgrps,omitempty"`
	Medias      []UserMedia `json:"medias,omitempty"`
}

// Users is an array of User
type Users []User

// UserGrpID is a reference to a user group by ID.
type UserGrpID struct {
	UserGroupID string `json:"usrgrpid"`
}

// UserMedia represents a media entry for a user.
type UserMedia struct {
	MediaTypeID string   `json:"mediatypeid"`
	SendTo      []string `json:"sendto"`
	Active      string   `json:"active"`
	Severity    string   `json:"severity"`
	Period      string   `json:"period"`
}

// UsersGet Wrapper for user.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/user/get
func (api *API) UsersGet(params Params) (res Users, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("user.get", params, &res)
	return
}

// UserGetByID Gets a user by ID only if there is exactly 1 matching user.
func (api *API) UserGetByID(id string) (res *User, err error) {
	users, err := api.UsersGet(Params{
		"userids":       id,
		"selectUsrgrps": "extend",
		"selectMedias":  "extend",
	})
	if err != nil {
		return
	}

	if len(users) == 1 {
		res = &users[0]
	} else {
		e := ExpectedOneResult(len(users))
		err = &e
	}
	return
}

// UsersCreate Wrapper for user.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/user/create
func (api *API) UsersCreate(users Users) (err error) {
	response, err := api.CallWithError("user.create", users)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["userids"].([]interface{})
	for i, id := range ids {
		users[i].UserID = id.(string)
	}
	return
}

// UsersUpdate Wrapper for user.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/user/update
func (api *API) UsersUpdate(users Users) (err error) {
	_, err = api.CallWithError("user.update", users)
	return
}

// UsersDeleteByIds Wrapper for user.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/user/delete
func (api *API) UsersDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("user.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	deleted := result["userids"].([]interface{})
	if len(ids) != len(deleted) {
		err = &ExpectedMore{len(ids), len(deleted)}
	}
	return
}
