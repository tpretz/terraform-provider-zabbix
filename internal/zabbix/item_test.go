package zabbix_test

import (
	"testing"

	dd "github.com/claranet/go-zabbix-api"
)

func CreateItem(app *dd.Application, t *testing.T) *dd.Item {
	items := dd.Items{{
		HostID:         app.HostID,
		Key:            "key.lala.laa",
		Name:           "name for key",
		Type:           dd.ZabbixTrapper,
		ApplicationIds: []string{app.ApplicationID},
	}}
	err := getAPI(t).ItemsCreate(items)
	if err != nil {
		t.Fatal(err)
	}
	return &items[0]
}

func DeleteItem(item *dd.Item, t *testing.T) {
	err := getAPI(t).ItemsDelete(dd.Items{*item})
	if err != nil {
		t.Fatal(err)
	}
}

func TestItems(t *testing.T) {
	api := getAPI(t)

	group := CreateHostGroup(t)
	defer DeleteHostGroup(group, t)

	host := CreateHost(group, t)
	defer DeleteHost(host, t)

	app := CreateApplication(host, t)
	defer DeleteApplication(app, t)

	items, err := api.ItemsGetByApplicationID(app.ApplicationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatal("Found items")
	}

	item := CreateItem(app, t)

	_, err = api.ItemGetByID(item.ItemID)
	if err != nil {
		t.Fatal(err)
	}

	item.Name = "another name"
	err = api.ItemsUpdate(dd.Items{*item})
	if err != nil {
		t.Error(err)
	}

	DeleteItem(item, t)
}
