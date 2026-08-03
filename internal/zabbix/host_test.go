package zabbix_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	zapi "github.com/tpretz/go-zabbix-api"
)

func CreateHost(group *zapi.HostGroup, t *testing.T) *zapi.Host {
	name := fmt.Sprintf("%s-%d", getHost(), rand.Int())
	iface := zapi.HostInterface{DNS: name, Port: "42", Type: zapi.Agent, UseIP: "0", Main: "1"}
	hosts := zapi.Hosts{{
		Host:       name,
		Name:       "Name for " + name,
		GroupIds:   zapi.HostGroupIDs{{group.GroupID}},
		Interfaces: zapi.HostInterfaces{iface},
	}}

	err := getAPI(t).HostsCreate(hosts)
	if err != nil {
		t.Fatal(err)
	}
	return &hosts[0]
}

func DeleteHost(host *zapi.Host, t *testing.T) {
	err := getAPI(t).HostsDelete(zapi.Hosts{*host})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHosts(t *testing.T) {
	api := getAPI(t)

	group := CreateHostGroup(t)
	defer DeleteHostGroup(group, t)

	hosts, err := api.HostsGetByHostGroups(zapi.HostGroups{*group})
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Errorf("Bad hosts: %#v", hosts)
	}

	host := CreateHost(group, t)
	if host.HostID == "" || host.Host == "" {
		t.Errorf("Something is empty: %#v", host)
	}
	host.GroupIds = nil
	host.Interfaces = nil
	host.ProxyID = "0"

	newName := fmt.Sprintf("%s-%d", getHost(), rand.Int())
	host.Host = newName
	err = api.HostsUpdate(zapi.Hosts{*host})
	if err != nil {
		t.Fatal(err)
	}

	host2, err := api.HostGetByHost(host.Host)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(host, host2) {
		t.Errorf("Hosts are not equal:\n%#v\n%#v", host, host2)
	}

	host2, err = api.HostGetByID(host.HostID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(host, host2) {
		t.Errorf("Hosts are not equal:\n%#v\n%#v", host, host2)
	}

	hosts, err = api.HostsGetByHostGroups(zapi.HostGroups{*group})
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 {
		t.Errorf("Bad hosts: %#v", hosts)
	}

	DeleteHost(host, t)

	hosts, err = api.HostsGetByHostGroups(zapi.HostGroups{*group})
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Errorf("Bad hosts: %#v", hosts)
	}
}
