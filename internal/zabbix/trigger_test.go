package zabbix_test

import (
	"fmt"
	"testing"

	dd "github.com/claranet/go-zabbix-api"
)

func CreateTrigger(item *dd.Item, host *dd.Host, t *testing.T) *dd.Trigger {
	expression := fmt.Sprintf("{%s:%s.last()}=0", host.Host, item.Key)
	triggers := dd.Triggers{{
		Description: "trigger description",
		Expression:  expression,
	}}
	err := getAPI(t).TriggersCreate(triggers)
	if err != nil {
		t.Fatal(err)
	}
	return &triggers[0]
}

func DeleteTrigger(trigger *dd.Trigger, t *testing.T) {
	err := getAPI(t).TriggersDelete(dd.Triggers{*trigger})
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

	triggerParam := dd.Params{"hostids": host.HostID}
	res, err := api.TriggersGet(triggerParam)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatal("Found items")
	}

	trigger := CreateTrigger(item, host, t)

	trigger.Description = "new trigger name"
	err = api.TriggersUpdate(dd.Triggers{*trigger})
	if err != nil {
		t.Error(err)
	}

	DeleteTrigger(trigger, t)
}
