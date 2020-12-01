package zabbix

// https://www.zabbix.com/documentation/5.0/manual/api/reference/host/object#host_inventory
type Inventory struct {
	Location string `json:"location,omitempty"`
	Model    string `json:"model,omitempty"`
	Name     string `json:"name,omitempty"`
	Notes    string `json:"notes,omitempty"`
}
