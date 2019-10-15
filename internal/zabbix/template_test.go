package zabbix_test

import (
	"testing"

	dd "github.com/claranet/go-zabbix-api"
)

func CreateTemplate(hostGroup *dd.HostGroup, t *testing.T) *dd.Template {
	template := dd.Templates{dd.Template{
		Host:   "template name",
		Groups: dd.HostGroups{*hostGroup},
	}}
	err := getAPI(t).TemplatesCreate(template)
	if err != nil {
		t.Fatal(err)
	}
	return &template[0]
}

func DeleteTemplate(template *dd.Template, t *testing.T) {
	err := getAPI(t).TemplatesDelete(dd.Templates{*template})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTemplates(t *testing.T) {
	api := getAPI(t)

	hostGroup := CreateHostGroup(t)
	defer DeleteHostGroup(hostGroup, t)

	templates, err := api.TemplatesGet(dd.Params{})
	if err != nil {
		t.Fatal(err)
	}

	if len(templates) == 0 {
		t.Fatal("No templates were obtained")
	}

	template := CreateTemplate(hostGroup, t)
	if template.TemplateID == "" {
		t.Errorf("Template id is empty %#v", template)
	}

	_, err = api.TemplateGetByID(template.TemplateID)
	if err != nil {
		t.Error(err)
	}

	template.Name = "new template name"
	err = api.TemplatesUpdate(dd.Templates{*template})
	if err != nil {
		t.Error(err)
	}

	DeleteTemplate(template, t)
}
