package zabbix

// MaintenanceType is whether data is still collected during the window
type MaintenanceType string

const (
	// MaintenanceWithData collect data during maintenance
	MaintenanceWithData MaintenanceType = "0"
	// MaintenanceNoData do not collect data during maintenance
	MaintenanceNoData MaintenanceType = "1"
)

// TimePeriodType is the recurrence of a maintenance period
type TimePeriodType string

const (
	TimePeriodOneTime TimePeriodType = "0"
	TimePeriodDaily   TimePeriodType = "2"
	TimePeriodWeekly  TimePeriodType = "3"
	TimePeriodMonthly TimePeriodType = "4"
)

// MaintenanceTimePeriod is a single period within a maintenance window.
// https://www.zabbix.com/documentation/7.0/manual/api/reference/maintenance/object#time-period
type MaintenanceTimePeriod struct {
	TimePeriodID   string         `json:"timeperiodid,omitempty"`
	TimePeriodType TimePeriodType `json:"timeperiod_type,omitempty"`
	Every          string         `json:"every,omitempty"`
	Month          string         `json:"month,omitempty"`
	DayOfWeek      string         `json:"dayofweek,omitempty"`
	Day            string         `json:"day,omitempty"`
	StartTime      string         `json:"start_time,omitempty"`
	StartDate      string         `json:"start_date,omitempty"`
	Period         string         `json:"period,omitempty"`
}

// MaintenanceTimePeriods is an array of MaintenanceTimePeriod
type MaintenanceTimePeriods []MaintenanceTimePeriod

// MaintenanceTag filters which problems are suppressed
type MaintenanceTag struct {
	Tag      string `json:"tag"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
}

// MaintenanceTags is an array of MaintenanceTag
type MaintenanceTags []MaintenanceTag

// Maintenance represents a Zabbix maintenance window.
//
// Note the asymmetry between write and read: host groups are supplied as
// "groups" but returned by selectHostGroups as "hostgroups".
// https://www.zabbix.com/documentation/7.0/manual/api/reference/maintenance/object
type Maintenance struct {
	MaintenanceID   string          `json:"maintenanceid,omitempty"`
	Name            string          `json:"name"`
	MaintenanceType MaintenanceType `json:"maintenance_type,omitempty"`
	Description     string          `json:"description,omitempty"`
	ActiveSince     string          `json:"active_since,omitempty"`
	ActiveTill      string          `json:"active_till,omitempty"`
	TagsEvalType    string          `json:"tags_evaltype,omitempty"`

	// write side
	GroupIDs []HostGroupID `json:"groups,omitempty"`
	HostIDs  []string      `json:"hosts,omitempty"`

	Tags        MaintenanceTags        `json:"tags,omitempty"`
	TimePeriods MaintenanceTimePeriods `json:"timeperiods,omitempty"`

	// read side, populated by selectHostGroups / selectHosts
	HostGroups HostGroups `json:"hostgroups,omitempty"`
	Hosts      Hosts      `json:"-"`
}

// Maintenances is an array of Maintenance
type Maintenances []Maintenance

// MaintenancesGet Wrapper for maintenance.get
// https://www.zabbix.com/documentation/7.0/manual/api/reference/maintenance/get
func (api *API) MaintenancesGet(params Params) (res Maintenances, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("maintenance.get", params, &res)
	return
}

// MaintenanceGetByID Gets a maintenance window by ID only if exactly 1 matches.
func (api *API) MaintenanceGetByID(id string) (res *Maintenance, err error) {
	items, err := api.MaintenancesGet(Params{"maintenanceids": id})
	if err != nil {
		return
	}
	if len(items) == 1 {
		res = &items[0]
	} else {
		e := ExpectedOneResult(len(items))
		err = &e
	}
	return
}

// MaintenancesCreate Wrapper for maintenance.create
// https://www.zabbix.com/documentation/7.0/manual/api/reference/maintenance/create
func (api *API) MaintenancesCreate(items Maintenances) (err error) {
	response, err := api.CallWithError("maintenance.create", items)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	ids := result["maintenanceids"].([]interface{})
	for i, id := range ids {
		items[i].MaintenanceID = id.(string)
	}
	return
}

// MaintenancesUpdate Wrapper for maintenance.update
// https://www.zabbix.com/documentation/7.0/manual/api/reference/maintenance/update
func (api *API) MaintenancesUpdate(items Maintenances) (err error) {
	_, err = api.CallWithError("maintenance.update", items)
	return
}

// MaintenancesDeleteByIds Wrapper for maintenance.delete
// https://www.zabbix.com/documentation/7.0/manual/api/reference/maintenance/delete
func (api *API) MaintenancesDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("maintenance.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	deleted := result["maintenanceids"].([]interface{})
	if len(ids) != len(deleted) {
		err = &ExpectedMore{len(ids), len(deleted)}
	}
	return
}
