package zabbix

import "testing"

func TestIsReadMethod(t *testing.T) {
	for m, want := range map[string]bool{
		"host.get": true, "item.get": true, "trigger.get": true,
		"apiinfo.version": true, "APIInfo.version": true,
		"host.create": false, "host.update": false, "host.delete": false,
		"host.massadd": false, "host.massupdate": false, "host.massremove": false,
		"template.massadd": false, "user.login": false, "trigger.create": false,
	} {
		if got := isReadMethod(m); got != want {
			t.Errorf("isReadMethod(%q) = %v, want %v", m, got, want)
		}
	}
}
