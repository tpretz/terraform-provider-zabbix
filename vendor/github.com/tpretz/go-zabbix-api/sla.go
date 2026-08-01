package zabbix

// SLAServiceTag represents a service tag used for SLA matching
type SLAServiceTag struct {
	Tag      string `json:"tag"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
}

// SLAServiceTags is an array of SLAServiceTag
type SLAServiceTags []SLAServiceTag

// SLASchedule represents a schedule entry for an SLA
type SLASchedule struct {
	PeriodFrom string `json:"period_from"`
	PeriodTo   string `json:"period_to"`
}

// SLASchedules is an array of SLASchedule
type SLASchedules []SLASchedule

// SLAExcludedDowntime represents an excluded downtime for an SLA
type SLAExcludedDowntime struct {
	Name       string `json:"name"`
	PeriodFrom string `json:"period_from"`
	PeriodTo   string `json:"period_to"`
}

// SLAExcludedDowntimes is an array of SLAExcludedDowntime
type SLAExcludedDowntimes []SLAExcludedDowntime

// SLA represents a Zabbix SLA object
// https://www.zabbix.com/documentation/7.0/manual/api/reference/sla/object
type SLA struct {
	SLAID             string               `json:"slaid,omitempty"`
	Name              string               `json:"name"`
	Period            string               `json:"period"`
	SLO               string               `json:"slo"`
	EffectiveDate     string               `json:"effective_date"`
	Timezone          string               `json:"timezone,omitempty"`
	Status            string               `json:"status,omitempty"`
	Description       string               `json:"description,omitempty"`
	ServiceTags       SLAServiceTags       `json:"service_tags,omitempty"`
	Schedule          SLASchedules         `json:"schedule,omitempty"`
	ExcludedDowntimes SLAExcludedDowntimes `json:"excluded_downtimes,omitempty"`
}

// SLAs is an array of SLA
type SLAs []SLA

// SLAsGet Wrapper for sla.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/sla/get
func (api *API) SLAsGet(params Params) (res SLAs, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("sla.get", params, &res)
	return
}

// SLAGetByID Gets an SLA by ID only if there is exactly 1 matching SLA.
func (api *API) SLAGetByID(id string) (res *SLA, err error) {
	slas, err := api.SLAsGet(Params{
		"slaids":                  id,
		"selectServiceTags":       "extend",
		"selectSchedule":          "extend",
		"selectExcludedDowntimes": "extend",
	})
	if err != nil {
		return
	}

	if len(slas) == 1 {
		res = &slas[0]
	} else {
		e := ExpectedOneResult(len(slas))
		err = &e
	}
	return
}

// SLAsCreate Wrapper for sla.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/sla/create
func (api *API) SLAsCreate(slas SLAs) (err error) {
	response, err := api.CallWithError("sla.create", slas)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	slaids := result["slaids"].([]interface{})
	for i, id := range slaids {
		slas[i].SLAID = id.(string)
	}
	return
}

// SLAsUpdate Wrapper for sla.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/sla/update
func (api *API) SLAsUpdate(slas SLAs) (err error) {
	_, err = api.CallWithError("sla.update", slas)
	return
}

// SLAsDeleteByIds Wrapper for sla.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/sla/delete
func (api *API) SLAsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("sla.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	slaids := result["slaids"].([]interface{})
	if len(ids) != len(slaids) {
		err = &ExpectedMore{len(ids), len(slaids)}
	}
	return
}
