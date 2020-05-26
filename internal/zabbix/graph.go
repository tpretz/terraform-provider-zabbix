package zabbix

type (
	GraphType string
	GraphAxis string

	GraphItemFunc string
	GraphItemDraw string
	GraphItemType string
	GraphItemSide string
)

const (
	GraphNormal   GraphType = "0"
	GraphStacked  GraphType = "1"
	GraphPie      GraphType = "2"
	GraphExploded GraphType = "3"

	GraphCalculated GraphAxis = "0"
	GraphFixed      GraphAxis = "1"
	GraphItem       GraphAxis = "2"

	GraphItemMin  GraphItemFunc = "1"
	GraphItemAvg  GraphItemFunc = "2"
	GraphItemMax  GraphItemFunc = "4"
	GraphItemAll  GraphItemFunc = "7"
	GraphItemLast GraphItemFunc = "9"

	GraphItemLine     GraphItemDraw = "0"
	GraphItemFilled   GraphItemDraw = "1"
	GraphItemBold     GraphItemDraw = "2"
	GraphItemDot      GraphItemDraw = "3"
	GraphItemDashed   GraphItemDraw = "4"
	GraphItemGradient GraphItemDraw = "5"

	GraphItemSimple GraphItemType = "0"
	GraphItemSum    GraphItemType = "2"

	GraphItemLeft  GraphItemSide = "0"
	GraphItemRight GraphItemSide = "1"
)

type GraphItem struct {
	GItemID   string        `json:"gitemid,omitempty"`
	GraphID   string        `json:"graphid,omitempty"`
	Color     string        `json:"color"`
	ItemID    string        `json:"itemid"`
	CalcFunc  GraphItemFunc `json:"calc_fnc,omitempty"`
	DrawType  GraphItemDraw `json:"drawtype,omitempty"`
	SortOrder string        `json:"sortorder,omitempty"`
	Type      GraphItemType `json:"type,omitempty"`
	YAxisSide GraphItemSide `json:"yaxisside,omitempty"`
}

type GraphItems []GraphItem

// Graph represent Zabbix Graph object
// https://www.zabbix.com/documentation/3.2/manual/api/reference/graph/object
type Graph struct {
	GraphID        string    `json:"graphid,omitempty"`
	Name           string    `json:"name"`
	Height         string    `json:"height"`
	Width          string    `json:"width"`
	GraphType      GraphType `json:"graphtype,omitempty"`
	PercentLeft    string    `json:"percent_left,omitempty"`
	PercentRight   string    `json:"percent_right,omitempty"`
	Show3d         string    `json:"show_3d,omitempty"`
	ShowLegend     string    `json:"show_legend,omitempty"`
	ShowWorkPeriod string    `json:"show_work_period,omitempty"`
	YAxisMax       string    `json:"yaxismax,omitempty"`
	YMaxItemId     string    `json:"ymax_itemid,omitempty"`
	YMaxType       string    `json:"ymax_type,omitempty"`
	YAxisMin       string    `json:"yaxismin,omitempty"`
	YMinItemId     string    `json:"ymin_itemid,omitempty"`
	YMinType       string    `json:"ymin_type,omitempty"`

	GItems GraphItems `json:"gitems,omitempty"`
}

// HostGroups is an array of HostGroup
type Graphs []Graph

// GraphsGet Wrapper for graph.get
// https://www.zabbix.com/documentation/3.2/manual/api/reference/graph/get
func (api *API) GraphsGet(params Params) (res Graphs, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("graph.get", params, &res)
	return
}
func (api *API) GraphProtosGet(params Params) (res Graphs, err error) {
	if _, present := params["output"]; !present {
		params["output"] = "extend"
	}
	err = api.CallWithErrorParse("graphprototype.get", params, &res)
	return
}

// GraphGetByID Gets host group by Id only if there is exactly 1 matching host group.
func (api *API) GraphGetByID(id string) (res *Graph, err error) {
	groups, err := api.GraphsGet(Params{"graphids": id})
	if err != nil {
		return
	}

	if len(groups) == 1 {
		res = &groups[0]
	} else {
		e := ExpectedOneResult(len(groups))
		err = &e
	}
	return
}
func (api *API) GraphProtoGetByID(id string) (res *Graph, err error) {
	groups, err := api.GraphProtosGet(Params{"graphids": id})
	if err != nil {
		return
	}

	if len(groups) == 1 {
		res = &groups[0]
	} else {
		e := ExpectedOneResult(len(groups))
		err = &e
	}
	return
}

// GraphsCreate Wrapper for graph.create
// https://www.zabbix.com/documentation/3.2/manual/api/reference/graph/create
func (api *API) GraphsCreate(hostGroups Graphs) (err error) {
	response, err := api.CallWithError("graph.create", hostGroups)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["graphids"].([]interface{})
	for i, id := range groupids {
		hostGroups[i].GraphID = id.(string)
	}
	return
}
func (api *API) GraphProtossCreate(hostGroups Graphs) (err error) {
	response, err := api.CallWithError("graphprototype.create", hostGroups)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["graphids"].([]interface{})
	for i, id := range groupids {
		hostGroups[i].GraphID = id.(string)
	}
	return
}

// GraphsUpdate Wrapper for graph.update
// https://www.zabbix.com/documentation/3.2/manual/api/reference/graph/update
func (api *API) GraphsUpdate(hostGroups Graphs) (err error) {
	_, err = api.CallWithError("graph.update", hostGroups)
	return
}
func (api *API) GraphProtosUpdate(hostGroups Graphs) (err error) {
	_, err = api.CallWithError("graphprototype.update", hostGroups)
	return
}

// HostGroupsDelete Wrapper for hostgroup.delete
// Cleans GroupId in all hostGroups elements if call succeed.
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/delete
func (api *API) GraphsDelete(hostGroups Graphs) (err error) {
	ids := make([]string, len(hostGroups))
	for i, group := range hostGroups {
		ids[i] = group.GraphID
	}

	err = api.GraphsDeleteByIds(ids)
	if err == nil {
		for i := range hostGroups {
			hostGroups[i].GraphID = ""
		}
	}
	return
}
func (api *API) GraphProtossDelete(hostGroups Graphs) (err error) {
	ids := make([]string, len(hostGroups))
	for i, group := range hostGroups {
		ids[i] = group.GraphID
	}

	err = api.GraphProtosDeleteByIds(ids)
	if err == nil {
		for i := range hostGroups {
			hostGroups[i].GraphID = ""
		}
	}
	return
}

// HostGroupsDeleteByIds Wrapper for hostgroup.delete
// https://www.zabbix.com/documentation/3.2/manual/api/reference/hostgroup/delete
func (api *API) GraphsDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("graph.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["graphids"].([]interface{})
	if len(ids) != len(groupids) {
		err = &ExpectedMore{len(ids), len(groupids)}
	}
	return
}
func (api *API) GraphProtosDeleteByIds(ids []string) (err error) {
	response, err := api.CallWithError("graphprototype.delete", ids)
	if err != nil {
		return
	}

	result := response.Result.(map[string]interface{})
	groupids := result["graphids"].([]interface{})
	if len(ids) != len(groupids) {
		err = &ExpectedMore{len(ids), len(groupids)}
	}
	return
}
