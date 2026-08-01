package zabbix

// ServiceTag represents a tag attached to a service
type ServiceTag struct {
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

// ServiceTags is an array of ServiceTag
type ServiceTags []ServiceTag

// ServiceProblemTag represents a problem tag used for matching events
type ServiceProblemTag struct {
	Tag      string `json:"tag"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
}

// ServiceProblemTags is an array of ServiceProblemTag
type ServiceProblemTags []ServiceProblemTag

// ServiceStatusRule represents a status rule for a service
type ServiceStatusRule struct {
	Type        string `json:"type"`
	LimitValue  string `json:"limit_value"`
	LimitStatus string `json:"limit_status"`
	NewStatus   string `json:"new_status"`
}

// ServiceStatusRules is an array of ServiceStatusRule
type ServiceStatusRules []ServiceStatusRule

// ServiceChild represents a child service link (only serviceid is used for create/update)
type ServiceChild struct {
	ServiceID string `json:"serviceid"`
}

// ServiceChildren is an array of ServiceChild
type ServiceChildren []ServiceChild

// ServiceParent represents a parent service link
type ServiceParent struct {
	ServiceID string `json:"serviceid"`
}

// ServiceParents is an array of ServiceParent
type ServiceParents []ServiceParent

// Service represents a Zabbix service object
// https://www.zabbix.com/documentation/7.0/manual/api/reference/service/object
type Service struct {
	ServiceID        string             `json:"serviceid,omitempty"`
	Name             string             `json:"name"`
	Algorithm        string             `json:"algorithm"`
	SortOrder        string             `json:"sortorder"`
	Weight           string             `json:"weight,omitempty"`
	PropagationRule  string             `json:"propagation_rule,omitempty"`
	PropagationValue string             `json:"propagation_value,omitempty"`
	Description      string             `json:"description,omitempty"`
	ProblemTags      ServiceProblemTags `json:"problem_tags,omitempty"`
	Tags             ServiceTags        `json:"tags,omitempty"`
	Children         ServiceChildren    `json:"children,omitempty"`
	Parents          ServiceParents     `json:"parents,omitempty"`
	StatusRules      ServiceStatusRules `json:"status_rules,omitempty"`

	// read-only
	UUID      string `json:"uuid,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Services is an array of Service
type Services []Service

// ServicesGet Wrapper for service.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/service/get
func (api *API) ServicesGet(params Params) (res Services, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("service.get", params, &res)
	return
}

// ServiceGetByID Gets a service by ID only if there is exactly 1 matching service.
func (api *API) ServiceGetByID(id string) (res *Service, err error) {
	services, err := api.ServicesGet(Params{
		"serviceids":        id,
		"selectProblemTags": "extend",
		"selectTags":        "extend",
		"selectChildren":    "extend",
		"selectParents":     "extend",
		"selectStatusRules": "extend",
	})
	if err != nil {
		return
	}

	if len(services) == 1 {
		res = &services[0]
	} else {
		e := ExpectedOneResult(len(services))
		err = &e
	}
	return
}

// ServicesCreate Wrapper for service.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/service/create
func (api *API) ServicesCreate(services Services) (err error) {
	response, err := api.CallWithError("service.create", services)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	serviceids := result["serviceids"].([]interface{})
	for i, id := range serviceids {
		services[i].ServiceID = id.(string)
	}
	return
}

// ServicesUpdate Wrapper for service.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/service/update
func (api *API) ServicesUpdate(services Services) (err error) {
	_, err = api.CallWithError("service.update", services)
	return
}

// ServicesDeleteByIds Wrapper for service.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/service/delete
func (api *API) ServicesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("service.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	serviceids := result["serviceids"].([]interface{})
	if len(ids) != len(serviceids) {
		err = &ExpectedMore{len(ids), len(serviceids)}
	}
	return
}
