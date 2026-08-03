package zabbix_test

import (
	"fmt"
	"testing"

	zapi "github.com/tpretz/go-zabbix-api"
)

func CreateTrigger(item *zapi.Item, host *zapi.Host, t *testing.T) *zapi.Trigger {
	expression := fmt.Sprintf("{%s:%s.last()}=0", host.Host, item.Key)
	triggers := zapi.Triggers{{
		Description: "trigger description",
		Expression:  expression,
	}}
	err := getAPI(t).TriggersCreate(triggers)
	if err != nil {
		t.Fatal(err)
	}
	return &triggers[0]
}

func DeleteTrigger(trigger *zapi.Trigger, t *testing.T) {
	err := getAPI(t).TriggersDelete(zapi.Triggers{*trigger})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTrigger(t *testing.T) {
	api := getAPI(t)

	group := CreateHostGroup(t)
	defer DeleteHostGroup(group, t)

	host := CreateHost(group, t)
	defer DeleteHost(host, t)

	app := CreateApplication(host, t)
	defer DeleteApplication(app, t)

	item := CreateItem(app, t)
	defer DeleteItem(item, t)

	triggerParam := zapi.Params{"hostids": host.HostID}
	res, err := api.TriggersGet(triggerParam)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatal("Found items")
	}

	trigger := CreateTrigger(item, host, t)

	trigger.Description = "new trigger name"
	err = api.TriggersUpdate(zapi.Triggers{*trigger})
	if err != nil {
		t.Error(err)
	}

	DeleteTrigger(trigger, t)
}
