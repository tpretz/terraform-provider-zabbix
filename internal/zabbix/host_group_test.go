package zabbix_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	dd "github.com/claranet/go-zabbix-api"
)

func CreateHostGroup(t *testing.T) *dd.HostGroup {
	hostGroups := dd.HostGroups{{Name: fmt.Sprintf("zabbix-testing-%d", rand.Int())}}
	err := getAPI(t).HostGroupsCreate(hostGroups)
	if err != nil {
		t.Fatal(err)
	}
	return &hostGroups[0]
}

func DeleteHostGroup(hostGroup *dd.HostGroup, t *testing.T) {
	err := getAPI(t).HostGroupsDelete(dd.HostGroups{*hostGroup})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHostGroups(t *testing.T) {
	api := getAPI(t)

	groups, err := api.HostGroupsGet(dd.Params{})
	if err != nil {
		t.Fatal(err)
	}

	hostGroup := CreateHostGroup(t)
	if hostGroup.GroupID == "" || hostGroup.Name == "" {
		t.Errorf("Something is empty: %#v", hostGroup)
	}

	hostGroup2, err := api.HostGroupGetByID(hostGroup.GroupID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(hostGroup, hostGroup2) {
		t.Errorf("Error getting group.\nOld group: %#v\nNew group: %#v", hostGroup, hostGroup2)
	}

	groups2, err := api.HostGroupsGet(dd.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups2) != len(groups)+1 {
		t.Errorf("Error creating group.\nOld groups: %#v\nNew groups: %#v", groups, groups2)
	}

	DeleteHostGroup(hostGroup, t)

	groups2, err = api.HostGroupsGet(dd.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(groups, groups2) {
		t.Errorf("Error deleting group.\nOld groups: %#v\nNew groups: %#v", groups, groups2)
	}
}
