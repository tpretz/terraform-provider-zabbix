package zabbix_test

import (
	"testing"

	zapi "github.com/tpretz/go-zabbix-api"
)

func CreateTemplate(hostGroup *zapi.HostGroup, t *testing.T) *zapi.Template {

	group := zapi.HostGroupID{
		GroupID: hostGroup.GroupID,
	}

	groups := []zapi.HostGroupID{group}

	template := zapi.Templates{zapi.Template{
		Host:   "template name",
		Groups: groups,
	}}
	err := getAPI(t).TemplatesCreate(template)
	if err != nil {
		t.Fatal(err)
	}
	return &template[0]
}

func DeleteTemplate(template *zapi.Template, t *testing.T) {
	err := getAPI(t).TemplatesDelete(zapi.Templates{*template})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTemplates(t *testing.T) {
	api := getAPI(t)

	hostGroup := CreateHostGroup(t)
	defer DeleteHostGroup(hostGroup, t)

	template := CreateTemplate(hostGroup, t)
	if template.TemplateID == "" {
		t.Errorf("Template id is empty %#v", template)
	}

	templates, err := api.TemplatesGet(zapi.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) == 0 {
		t.Fatal("No templates were obtained")
	}

	_, err = api.TemplateGetByID(template.TemplateID)
	if err != nil {
		t.Error(err)
	}

	template.Name = "new template name"
	err = api.TemplatesUpdate(zapi.Templates{*template})
	if err != nil {
		t.Error(err)
	}

	DeleteTemplate(template, t)
}
