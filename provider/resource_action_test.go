package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tpretz/go-zabbix-api"
)

func TestAccAction_TriggerMessage(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckActionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_action" "trigger" {
  name        = "acc-action-trigger-%s"
  eventsource = "trigger"
  status      = "disabled"
  esc_period  = "60s"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "1"
      operator      = "0"
      value         = "10084"
    }
  }

  operations {
    operationtype = "0"
    esc_step_from = "1"
    esc_step_to   = "1"
    esc_period    = "0"

    opmessage {
      default_msg  = "1"
      mediatypeid  = "0"
    }
    opmessage_grp {
      usrgrpid = "7"
    }
  }
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_action.trigger", "name", "acc-action-trigger-"+id),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "eventsource", "trigger"),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "status", "disabled"),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "esc_period", "60s"),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "operations.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_action" "trigger" {
  name        = "acc-action-trigger-updated-%s"
  eventsource = "trigger"
  status      = "disabled"
  esc_period  = "120s"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "1"
      operator      = "0"
      value         = "10084"
    }
  }

  operations {
    operationtype = "0"
    esc_step_from = "1"
    esc_step_to   = "2"
    esc_period    = "0"

    opmessage {
      default_msg  = "1"
      mediatypeid  = "0"
    }
    opmessage_grp {
      usrgrpid = "7"
    }
  }

  recovery_operations {
    operationtype = "11"

    opmessage {
      default_msg = "1"
    }
  }

  update_operations {
    operationtype = "12"

    opmessage {
      default_msg = "1"
    }
  }
}
`, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_action.trigger", "name", "acc-action-trigger-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "esc_period", "120s"),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "operations.0.esc_step_to", "2"),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "recovery_operations.#", "1"),
					resource.TestCheckResourceAttr("zabbix_action.trigger", "update_operations.#", "1"),
				),
			},
		},
	})
}

func TestAccAction_TriggerRemoteCommand(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckActionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_script" "for_action" {
  name       = "acc-script-for-action-%s"
  command    = "echo hello"
  type       = "script"
  scope      = "action_operation"
  execute_on = "server"
}

resource "zabbix_action" "cmd" {
  name        = "acc-action-cmd-%s"
  eventsource = "trigger"
  status      = "disabled"
  esc_period  = "60s"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "1"
      operator      = "0"
      value         = "10084"
    }
  }

  operations {
    operationtype = "1"
    esc_step_from = "1"
    esc_step_to   = "1"
    esc_period    = "0"

    opcommand {
      scriptid = zabbix_script.for_action.id
    }
    opcommand_hst {
      hostid = "0"
    }
  }
}
`, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_action.cmd", "name", "acc-action-cmd-"+id),
					resource.TestCheckResourceAttr("zabbix_action.cmd", "operations.0.operationtype", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_script" "for_action" {
  name       = "acc-script-for-action-%s"
  command    = "echo updated"
  type       = "script"
  scope      = "action_operation"
  execute_on = "server"
}

resource "zabbix_action" "cmd" {
  name        = "acc-action-cmd-updated-%s"
  eventsource = "trigger"
  status      = "disabled"
  esc_period  = "120s"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "1"
      operator      = "0"
      value         = "10084"
    }
  }

  operations {
    operationtype = "1"
    esc_step_from = "1"
    esc_step_to   = "2"
    esc_period    = "0"

    opcommand {
      scriptid = zabbix_script.for_action.id
    }
    opcommand_hst {
      hostid = "0"
    }
  }
}
`, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_action.cmd", "name", "acc-action-cmd-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_action.cmd", "esc_period", "120s"),
					resource.TestCheckResourceAttr("zabbix_action.cmd", "operations.0.esc_step_to", "2"),
				),
			},
		},
	})
}

func TestAccAction_Autoregistration(t *testing.T) {
	id := resource.UniqueId()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckActionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "for_action" {
  name = "acc-group-for-action-%s"
}

resource "zabbix_action" "autoreg" {
  name        = "acc-action-autoreg-%s"
  eventsource = "autoregistration"
  status      = "disabled"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "22"
      operator      = "2"
      value         = "linux"
    }
  }

  operations {
    operationtype = "4"

    opgroup {
      groupid = zabbix_hostgroup.for_action.id
    }
  }
}
`, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_action.autoreg", "name", "acc-action-autoreg-"+id),
					resource.TestCheckResourceAttr("zabbix_action.autoreg", "eventsource", "autoregistration"),
					resource.TestCheckResourceAttr("zabbix_action.autoreg", "operations.0.operationtype", "4"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "zabbix_hostgroup" "for_action" {
  name = "acc-group-for-action-%s"
}

resource "zabbix_action" "autoreg" {
  name        = "acc-action-autoreg-updated-%s"
  eventsource = "autoregistration"
  status      = "disabled"

  filter {
    evaltype = "and_or"
    conditions {
      conditiontype = "22"
      operator      = "2"
      value         = "windows"
    }
  }

  operations {
    operationtype = "4"

    opgroup {
      groupid = zabbix_hostgroup.for_action.id
    }
  }
}
`, id, id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_action.autoreg", "name", "acc-action-autoreg-updated-"+id),
					resource.TestCheckResourceAttr("zabbix_action.autoreg", "filter.0.conditions.0.value", "windows"),
				),
			},
		},
	})
}

func testAccCheckActionDestroy(s *terraform.State) error {
	api := testAccProvider.Meta().(*zabbix.API)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "zabbix_action" {
			continue
		}
		action, err := api.ActionGetByID(rs.Primary.ID)
		if err == nil && action != nil {
			return fmt.Errorf("action %s still exists", rs.Primary.ID)
		}
	}
	return nil
}
