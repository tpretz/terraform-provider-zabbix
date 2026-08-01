package zabbix

// MediaType represents a Zabbix media type object.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/mediatype/object
type MediaType struct {
	MediaTypeID        string                     `json:"mediatypeid,omitempty"`
	Name               string                     `json:"name"`
	Type               string                     `json:"type"`
	SmtpServer         string                     `json:"smtp_server,omitempty"`
	SmtpHelo           string                     `json:"smtp_helo,omitempty"`
	SmtpEmail          string                     `json:"smtp_email,omitempty"`
	SmtpPort           string                     `json:"smtp_port,omitempty"`
	SmtpSecurity       string                     `json:"smtp_security,omitempty"`
	SmtpVerifyPeer     string                     `json:"smtp_verify_peer,omitempty"`
	SmtpVerifyHost     string                     `json:"smtp_verify_host,omitempty"`
	SmtpAuthentication string                     `json:"smtp_authentication,omitempty"`
	Username           string                     `json:"username,omitempty"`
	Passwd             string                     `json:"passwd,omitempty"`
	ExecPath           string                     `json:"exec_path,omitempty"`
	ExecParams         string                     `json:"exec_params,omitempty"`
	GsmModem           string                     `json:"gsm_modem,omitempty"`
	Script             string                     `json:"script,omitempty"`
	Timeout            string                     `json:"timeout,omitempty"`
	ProcessTags        string                     `json:"process_tags,omitempty"`
	ShowEventMenu      string                     `json:"show_event_menu,omitempty"`
	EventMenuURL       string                     `json:"event_menu_url,omitempty"`
	EventMenuName      string                     `json:"event_menu_name,omitempty"`
	Status             string                     `json:"status,omitempty"`
	Description        string                     `json:"description,omitempty"`
	MaxSessions        string                     `json:"maxsessions,omitempty"`
	MaxAttempts        string                     `json:"maxattempts,omitempty"`
	AttemptInterval    string                     `json:"attempt_interval,omitempty"`
	ContentType        string                     `json:"content_type,omitempty"`
	MessageFormat      string                     `json:"message_format,omitempty"`
	MessageTemplates   []MediaTypeMessageTemplate `json:"message_templates,omitempty"`
	Parameters         []MediaTypeParameter       `json:"parameters,omitempty"`
}

// MediaTypeMessageTemplate represents a message template for a media type.
type MediaTypeMessageTemplate struct {
	EventSource string `json:"eventsource"`
	Recovery    string `json:"recovery"`
	Subject     string `json:"subject,omitempty"`
	Message     string `json:"message,omitempty"`
}

// MediaTypeParameter represents a webhook parameter for a media type.
type MediaTypeParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MediaTypes is an array of MediaType
type MediaTypes []MediaType

// MediaTypesGet Wrapper for mediatype.get
func (api *API) MediaTypesGet(params Params) (res MediaTypes, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("mediatype.get", params, &res)
	return
}

// MediaTypeGetByID Gets a media type by ID only if there is exactly 1 matching media type.
func (api *API) MediaTypeGetByID(id string) (res *MediaType, err error) {
	mts, err := api.MediaTypesGet(Params{"mediatypeids": id})
	if err != nil {
		return
	}

	if len(mts) == 1 {
		res = &mts[0]
	} else {
		e := ExpectedOneResult(len(mts))
		err = &e
	}
	return
}

// MediaTypesCreate Wrapper for mediatype.create
func (api *API) MediaTypesCreate(mediaTypes MediaTypes) (err error) {
	response, err := api.CallWithError("mediatype.create", mediaTypes)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["mediatypeids"].([]interface{})
	for i, id := range ids {
		mediaTypes[i].MediaTypeID = id.(string)
	}
	return
}

// MediaTypesUpdate Wrapper for mediatype.update
func (api *API) MediaTypesUpdate(mediaTypes MediaTypes) (err error) {
	_, err = api.CallWithError("mediatype.update", mediaTypes)
	return
}

// MediaTypesDeleteByIds Wrapper for mediatype.delete
func (api *API) MediaTypesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("mediatype.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	mediatypeids := result["mediatypeids"].([]interface{})
	if len(ids) != len(mediatypeids) {
		err = &ExpectedMore{len(ids), len(mediatypeids)}
	}
	return
}
