package zabbix_test

import (
	"testing"

	zapi "github.com/tpretz/go-zabbix-api"
)

func CreateItem(app *zapi.Application, t *testing.T) *zapi.Item {
	items := zapi.Items{{
		HostID:         app.HostID,
		Key:            "key.lala.laa",
		Name:           "name for key",
		Type:           zapi.ZabbixTrapper,
		ApplicationIds: []string{app.ApplicationID},
	}}
	err := getAPI(t).ItemsCreate(items)
	if err != nil {
		t.Fatal(err)
	}
	return &items[0]
}

func DeleteItem(item *zapi.Item, t *testing.T) {
	err := getAPI(t).ItemsDelete(zapi.Items{*item})
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
	err = api.ItemsUpdate(zapi.Items{*item})
	if err != nil {
		t.Error(err)
	}

	DeleteItem(item, t)
}
