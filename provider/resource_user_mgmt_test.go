package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccUsergroup(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUsergroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUsergroupConfig(id, "default", true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "name", "tf-acc-ug-"+id),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "gui_access", "default"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "users_status", "true"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "debug_mode", "false"),
				),
			},
			{
				// Update: change name, gui_access, debug_mode, add permissions
				Config: testAccUsergroupConfigUpdated(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "name", "tf-acc-ug-renamed-"+id),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "gui_access", "internal"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "users_status", "true"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "debug_mode", "true"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "hostgroup_rights.#", "1"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "hostgroup_rights.0.permission", "read-write"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "templategroup_rights.#", "1"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "templategroup_rights.0.permission", "read-only"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "tag_filters.#", "1"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "tag_filters.0.tag", "env"),
					resource.TestCheckResourceAttr("zabbix_usergroup.test", "tag_filters.0.value", "prod"),
				),
			},
		},
	})
}

func TestAccDataUsergroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "zabbix_usergroup" "test" {
  name = "Zabbix administrators"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zabbix_usergroup.test", "id"),
					resource.TestCheckResourceAttr("data.zabbix_usergroup.test", "name", "Zabbix administrators"),
				),
			},
		},
	})
}

func testAccUsergroupConfig(id, guiAccess string, enabled, debug bool) string {
	return fmt.Sprintf(`
resource "zabbix_usergroup" "test" {
  name         = "tf-acc-ug-%s"
  gui_access   = "%s"
  users_status = %t
  debug_mode   = %t
}
`, id, guiAccess, enabled, debug)
}

func testAccUsergroupConfigUpdated(id string) string {
	return fmt.Sprintf(`
data "zabbix_hostgroup" "linux" {
  name = "Linux servers"
}

data "zabbix_templategroup" "templates" {
  name = "Templates"
}

resource "zabbix_usergroup" "test" {
  name         = "tf-acc-ug-renamed-%s"
  gui_access   = "internal"
  users_status = true
  debug_mode   = true

  hostgroup_rights {
    id         = data.zabbix_hostgroup.linux.id
    permission = "read-write"
  }

  templategroup_rights {
    id         = data.zabbix_templategroup.templates.id
    permission = "read-only"
  }

  tag_filters {
    groupid = data.zabbix_hostgroup.linux.id
    tag     = "env"
    value   = "prod"
  }
}
`, id)
}

func testAccCheckUsergroupDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_usergroup" {
			continue
		}
		groups, err := api.UserGroupsGet(zabbix.Params{"usrgrpids": rs.Primary.ID})
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			return fmt.Errorf("usergroup %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

// --- Role Tests ---

func TestAccRole(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_role.test", "name", "tf-acc-role-"+id),
					resource.TestCheckResourceAttr("zabbix_role.test", "type", "user"),
					resource.TestCheckResourceAttr("zabbix_role.test", "ui_default_access", "true"),
					resource.TestCheckResourceAttr("zabbix_role.test", "actions_default_access", "true"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_access", "true"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_mode", "deny"),
				),
			},
			{
				// Update: change name, add UI and action rules, switch api_mode
				Config: testAccRoleConfigUpdated(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_role.test", "name", "tf-acc-role-renamed-"+id),
					resource.TestCheckResourceAttr("zabbix_role.test", "type", "user"),
					resource.TestCheckResourceAttr("zabbix_role.test", "ui_default_access", "false"),
					resource.TestCheckResourceAttr("zabbix_role.test", "actions_default_access", "false"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_access", "true"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_mode", "allow"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_methods.#", "2"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_methods.0", "host.get"),
					resource.TestCheckResourceAttr("zabbix_role.test", "api_methods.1", "item.get"),
				),
			},
		},
	})
}

func TestAccDataRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "zabbix_role" "test" {
  name = "Super admin role"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.zabbix_role.test", "id"),
					resource.TestCheckResourceAttr("data.zabbix_role.test", "name", "Super admin role"),
					resource.TestCheckResourceAttr("data.zabbix_role.test", "type", "super_admin"),
				),
			},
		},
	})
}

func testAccRoleConfig(id string) string {
	return fmt.Sprintf(`
resource "zabbix_role" "test" {
  name               = "tf-acc-role-%s"
  type               = "user"
  ui_default_access  = true
  actions_default_access = true
  api_access         = true
  api_mode           = "deny"
}
`, id)
}

func testAccRoleConfigUpdated(id string) string {
	return fmt.Sprintf(`
resource "zabbix_role" "test" {
  name               = "tf-acc-role-renamed-%s"
  type               = "user"
  ui_default_access  = false
  actions_default_access = false
  api_access         = true
  api_mode           = "allow"

  ui {
    name   = "monitoring.dashboard"
    status = true
  }

  actions {
    name   = "edit_dashboards"
    status = true
  }

  api_methods = ["host.get", "item.get"]
}
`, id)
}
func testAccCheckRoleDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_role" {
			continue
		}
		roles, err := api.RolesGet(zabbix.Params{"roleids": rs.Primary.ID})
		if err != nil {
			return err
		}
		if len(roles) > 0 {
			return fmt.Errorf("role %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

// --- User Tests ---

func TestAccUser(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_user.test", "username", "tf-acc-user-"+id),
					resource.TestCheckResourceAttr("zabbix_user.test", "name", "Test"),
					resource.TestCheckResourceAttr("zabbix_user.test", "surname", "User"),
					resource.TestCheckResourceAttr("zabbix_user.test", "autologin", "false"),
					resource.TestCheckResourceAttrSet("zabbix_user.test", "roleid"),
				),
			},
			{
				// Update: change name, surname, add media
				Config: testAccUserConfigUpdated(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_user.test", "username", "tf-acc-user-renamed-"+id),
					resource.TestCheckResourceAttr("zabbix_user.test", "name", "Updated"),
					resource.TestCheckResourceAttr("zabbix_user.test", "surname", "Person"),
					resource.TestCheckResourceAttr("zabbix_user.test", "autologin", "true"),
					resource.TestCheckResourceAttr("zabbix_user.test", "medias.#", "1"),
					resource.TestCheckResourceAttr("zabbix_user.test", "medias.0.mediatypeid", "1"),
					resource.TestCheckResourceAttr("zabbix_user.test", "medias.0.active", "true"),
				),
			},
		},
	})
}

func testAccUserConfig(id string) string {
	return fmt.Sprintf(`
resource "zabbix_usergroup" "test_for_user" {
  name = "tf-acc-ug-for-user-%s"
}

resource "zabbix_user" "test" {
  username = "tf-acc-user-%s"
  name     = "Test"
  surname  = "User"
  passwd   = "Zabbix12345!"
  roleid   = "1"
  usrgrps  = [zabbix_usergroup.test_for_user.id]
}
`, id, id)
}

func testAccUserConfigUpdated(id string) string {
	return fmt.Sprintf(`
resource "zabbix_usergroup" "test_for_user" {
  name = "tf-acc-ug-for-user-%s"
}

resource "zabbix_user" "test" {
  username  = "tf-acc-user-renamed-%s"
  name      = "Updated"
  surname   = "Person"
  passwd    = "Zabbix12345!"
  roleid    = "1"
  usrgrps   = [zabbix_usergroup.test_for_user.id]
  autologin = true
  autologout = "0"

  medias {
    mediatypeid = "1"
    sendto      = ["test@example.com"]
    active      = true
    severity    = 63
    period      = "1-7,00:00-24:00"
  }
}
`, id, id)
}

func testAccCheckUserDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_user" {
			continue
		}
		users, err := api.UsersGet(zabbix.Params{"userids": rs.Primary.ID})
		if err != nil {
			return err
		}
		if len(users) > 0 {
			return fmt.Errorf("user %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
