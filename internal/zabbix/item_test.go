package zabbix_test

import (
	"testing"

	zapi "github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

func CreateItem(host *zapi.Host, t *testing.T) *zapi.Item {
	items := zapi.Items{{
		HostID: host.HostID,
		Key:    "key.lala.laa",
		Name:   "name for key",
		Type:   zapi.ZabbixTrapper,
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

	item := CreateItem(host, t)

	_, err := api.ItemGetByID(item.ItemID)
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
