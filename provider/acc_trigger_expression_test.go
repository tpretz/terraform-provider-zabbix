package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Trigger expression round trips.
//
// Zabbix stores a trigger expression with every item reference replaced by a
// {functionid} token, and trigger.get will undo that -- but only with
// "expandExpression", which also renders user macros down to their values.
// Reading through that flag put a value into state where the configuration
// held a macro, so the two could never agree:
//
//	config  min(/host/net.tcp.service.perf[https,,443],5m)>{$HTTPS.RESPONSE.SLOW}
//	state   min(/host/net.tcp.service.perf[https,,443],5m)>5
//
// Every plan then proposed rewriting the expression, applied, and proposed it
// again. The provider now rebuilds the source form from the functions, items
// and hosts instead (trigger_expression.go), and these are the tests that hold
// that against real servers.
//
// The assertion in every step is **an empty plan after apply**, which the test
// framework makes for us, plus state matching the configuration byte for byte.
// Both matter: an expression that merely plans clean could still have been
// mangled in some way both the read and the write agree on.
//
// The grammar is covered exhaustively and cheaply by the unit tests in
// trigger_expression_test.go. What is here is what only a live server can
// answer -- that Zabbix stores these expressions the way the probe said it
// does, on every supported version, and that what comes back out is what went
// in.

// triggerExprFixtureHCL supplies two templates so that an expression can span
// hosts, macros on each, and items of both a numeric and a text type.
//
// Two templates rather than two hosts: Zabbix refuses an expression whose
// elements "belong to a template and a host simultaneously", and templates are
// what the rest of this suite's trigger fixtures already use.
const triggerExprFixtureHCL = `
resource "zabbix_templategroup" "testtrigexprgrp" {
	name = "test-trigexpr-group"
}
resource "zabbix_template" "testtrigexpra" {
	groups = [ zabbix_templategroup.testtrigexprgrp.id ]
	host = "test-trigexpr-a"

	macro {
		name = "{$SLOW}"
		value = "5"
	}
	macro {
		name = "{$WIN}"
		value = "5m"
	}
}
resource "zabbix_template" "testtrigexprb" {
	groups = [ zabbix_templategroup.testtrigexprgrp.id ]
	host = "test-trigexpr-b"

	macro {
		name = "{$FAST}"
		value = "1"
	}
}
resource "zabbix_item_trapper" "testtrigexpritema" {
	hostid = zabbix_template.testtrigexpra.id
	key = "trigexpr.a"

	name = "Trigger Expression A"
	valuetype = "float"
}
resource "zabbix_item_trapper" "testtrigexpritema2" {
	hostid = zabbix_template.testtrigexpra.id
	key = "trigexpr.a2"

	name = "Trigger Expression A2"
	valuetype = "float"
}
resource "zabbix_item_trapper" "testtrigexpritemtxt" {
	hostid = zabbix_template.testtrigexpra.id
	key = "trigexpr.txt"

	name = "Trigger Expression Text"
	valuetype = "text"
}
resource "zabbix_item_trapper" "testtrigexpritemb" {
	hostid = zabbix_template.testtrigexprb.id
	key = "trigexpr.b"

	name = "Trigger Expression B"
	valuetype = "float"
}
`

// triggerExprHCL wraps a zabbix_trigger body in the fixture above.
func triggerExprHCL(body string) string {
	return triggerExprFixtureHCL + `
resource "zabbix_trigger" "testtrigexpr" {
	name = "Test Trigger Expression"

	depends_on = [
		zabbix_item_trapper.testtrigexpritema,
		zabbix_item_trapper.testtrigexpritema2,
		zabbix_item_trapper.testtrigexpritemtxt,
		zabbix_item_trapper.testtrigexpritemb,
	]
` + body + `}
`
}

// exprStep builds one round-trip step: apply the expression, assert state holds
// exactly what the configuration said, and assert the server kept the macros
// the configuration named. The framework's own empty-plan check after the step
// is the assertion the bug report is about.
func exprStep(t *testing.T, expression string, storedMacros ...string) resource.TestStep {
	const addr = "zabbix_trigger.testtrigexpr"
	return resource.TestStep{
		Config: hcl(t, triggerExprHCL("\texpression = "+quoteHCL(expression)+"\n")),
		Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(addr, "expression", expression),
			testAccCheckServerAttrContains(addr, serverTriggerRaw, "expression", storedMacros...),
		),
	}
}

func TestAccTriggerExpressionRoundTrip(t *testing.T) {
	const addr = "zabbix_trigger.testtrigexpr"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			// the reported configuration's shape: a user macro on the right
			// hand side of the comparison
			exprStep(t, `min(/test-trigexpr-a/trigexpr.a,5m)>{$SLOW}`, `{$SLOW}`),

			// a user macro inside a function parameter, which is the half of
			// the problem the raw expression alone does not show -- the macro
			// lives in the functions table, not in the expression
			exprStep(t, `min(/test-trigexpr-a/trigexpr.a,{$WIN})>{$SLOW}`, `{$SLOW}`),

			// several functions over several items, two of them the same
			// function over different items, so element identity has to come
			// from the functionid and not from position
			exprStep(t,
				`min(/test-trigexpr-a/trigexpr.a,{$WIN})>{$SLOW} and last(/test-trigexpr-a/trigexpr.a2)=0 and last(/test-trigexpr-a/trigexpr.a)<>last(/test-trigexpr-a/trigexpr.a2)`,
				`{$SLOW}`),

			// spanning two templates. The trigger's own host list has two
			// entries here and says nothing about which item lives where, so
			// this is the step that fails if item -> host is resolved through
			// it rather than through each item's hostid.
			exprStep(t,
				`last(/test-trigexpr-a/trigexpr.a)>{$SLOW} and last(/test-trigexpr-b/trigexpr.b)<{$FAST}`,
				`{$SLOW}`, `{$FAST}`),

			// quoting, empty parameters, whitespace the user chose, and a
			// function that reads no item at all and so has no functionid
			exprStep(t,
				`count(/test-trigexpr-a/trigexpr.txt,1h,,"error,x")>{$SLOW} and last( /test-trigexpr-a/trigexpr.a ) > 0 and date()>20200101`,
				`{$SLOW}`),

			{ // and it is stable: the plan on an unchanged configuration is
				// empty, which is the whole of the bug
				Config: hcl(t, triggerExprHCL("\texpression = "+quoteHCL(
					`count(/test-trigexpr-a/trigexpr.txt,1h,,"error,x")>{$SLOW} and last( /test-trigexpr-a/trigexpr.a ) > 0 and date()>20200101`)+"\n")),
				PlanOnly: true,
			},
			{ // import: the reconstruction has to agree with itself when the
				// resource is read with no prior state
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccTriggerExpressionRecovery does the same for recovery_expression, which
// had the same defect for the same reason and needed the same fix. Its
// functions come back in the same selectFunctions array as the problem
// expression's, so the step that matters is the one where the two expressions
// name different items on different templates.
func TestAccTriggerExpressionRecovery(t *testing.T) {
	const addr = "zabbix_trigger.testtrigexpr"

	step := func(expression, recovery string, stored ...string) resource.TestStep {
		return resource.TestStep{
			Config: hcl(t, triggerExprHCL(
				"\texpression = "+quoteHCL(expression)+"\n"+
					"\trecovery_expression = "+quoteHCL(recovery)+"\n")),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(addr, "expression", expression),
				resource.TestCheckResourceAttr(addr, "recovery_expression", recovery),
				testAccCheckServerAttrContains(addr, serverTriggerRaw, "recovery_expression", stored...),
			),
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			step(
				`min(/test-trigexpr-a/trigexpr.a,{$WIN})>{$SLOW}`,
				`max(/test-trigexpr-a/trigexpr.a,{$WIN})<{$SLOW}`,
				`{$SLOW}`),
			step(
				// the recovery expression names an item, and a template, that
				// the problem expression does not
				`last(/test-trigexpr-a/trigexpr.a)>{$SLOW}`,
				`last(/test-trigexpr-a/trigexpr.a2)<1 and last(/test-trigexpr-b/trigexpr.b)<{$FAST}`,
				`{$FAST}`),
			{
				Config: hcl(t, triggerExprHCL(
					"\texpression = "+quoteHCL(`last(/test-trigexpr-a/trigexpr.a)>{$SLOW}`)+"\n"+
						"\trecovery_expression = "+quoteHCL(`last(/test-trigexpr-a/trigexpr.a2)<1 and last(/test-trigexpr-b/trigexpr.b)<{$FAST}`)+"\n")),
				PlanOnly: true,
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// protoTriggerExprFixtureHCL is the prototype equivalent: a discovery rule and
// two item prototypes, so that an expression can carry an LLD macro and a user
// macro at the same time.
const protoTriggerExprFixtureHCL = `
resource "zabbix_templategroup" "testtrigexprgrp" {
	name = "test-trigexpr-group"
}
resource "zabbix_template" "testtrigexpra" {
	groups = [ zabbix_templategroup.testtrigexprgrp.id ]
	host = "test-trigexpr-a"

	macro {
		name = "{$SLOW}"
		value = "5"
	}
	macro {
		name = "{$WIN}"
		value = "5m"
	}
}
resource "zabbix_lld_trapper" "testtrigexprlld" {
	hostid = zabbix_template.testtrigexpra.id
	key = "trigexpr.lld"
	name = "Trigger Expression LLD"
	delay = "0"
}
resource "zabbix_proto_item_trapper" "testtrigexprproto" {
	hostid = zabbix_template.testtrigexpra.id
	ruleid = zabbix_lld_trapper.testtrigexprlld.id
	key = "trigexpr[{#FSNAME}]"

	name = "Trigger Expression Proto {#FSNAME}"
	valuetype = "float"
}
resource "zabbix_proto_item_trapper" "testtrigexprproto2" {
	hostid = zabbix_template.testtrigexpra.id
	ruleid = zabbix_lld_trapper.testtrigexprlld.id
	key = "trigexpr2[{#FSNAME}]"

	name = "Trigger Expression Proto 2 {#FSNAME}"
	valuetype = "float"
}
`

func protoTriggerExprHCL(body string) string {
	return protoTriggerExprFixtureHCL + `
resource "zabbix_proto_trigger" "testtrigexpr" {
	name = "Test Proto Trigger Expression {#FSNAME}"

	depends_on = [
		zabbix_proto_item_trapper.testtrigexprproto,
		zabbix_proto_item_trapper.testtrigexprproto2,
	]
` + body + `}
`
}

// TestAccProtoTriggerExpressionRoundTrip is the prototype half. A trigger
// prototype's expression carries {#LLD} macros as well as {$USER} ones, in the
// item key and in function parameters both, and all of them have to survive the
// same read.
func TestAccProtoTriggerExpressionRoundTrip(t *testing.T) {
	const addr = "zabbix_proto_trigger.testtrigexpr"

	step := func(expression string, stored ...string) resource.TestStep {
		return resource.TestStep{
			Config: hcl(t, protoTriggerExprHCL("\texpression = "+quoteHCL(expression)+"\n")),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(addr, "expression", expression),
				testAccCheckServerAttrContains(addr, serverProtoTriggerRaw, "expression", stored...),
			),
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			// an LLD macro in the item key and a user macro in the comparison
			step(`last(/test-trigexpr-a/trigexpr[{#FSNAME}])>{$SLOW}`, `{$SLOW}`),

			// a user macro in a function parameter as well
			step(`min(/test-trigexpr-a/trigexpr[{#FSNAME}],{$WIN})>{$SLOW}`, `{$SLOW}`),

			// an LLD macro *inside* a function parameter, which is where it is
			// least visible: it is stored in the functions table, not in the
			// expression, so nothing about the raw expression shows it
			step(
				`count(/test-trigexpr-a/trigexpr[{#FSNAME}],{$WIN},,"{#VALUE}")>{$SLOW} and last(/test-trigexpr-a/trigexpr2[{#FSNAME}])>0`,
				`{$SLOW}`),
			{
				Config: hcl(t, protoTriggerExprHCL("\texpression = "+quoteHCL(
					`count(/test-trigexpr-a/trigexpr[{#FSNAME}],{$WIN},,"{#VALUE}")>{$SLOW} and last(/test-trigexpr-a/trigexpr2[{#FSNAME}])>0`)+"\n")),
				PlanOnly: true,
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// serverTriggerRaw and serverProtoTriggerRaw read a trigger *without*
// "expandExpression", so the expression comes back in the form Zabbix actually
// stores: item references as {functionid}, and every macro exactly as the user
// wrote it. That is the only place a macro can be seen to have survived --
// state is written by the provider's own read, and the expanded form has the
// macro gone by definition.
var (
	serverTriggerRaw      = serverObject("trigger.get", "triggerids", nil)
	serverProtoTriggerRaw = serverObject("triggerprototype.get", "triggerids", nil)
)

// testAccCheckServerAttrContains re-reads an object from Zabbix and asserts a
// property contains each of the given substrings. Containment rather than
// equality because the stored expression is full of server-assigned function
// ids, which no fixture can predict.
func testAccCheckServerAttrContains(addr string, get serverGetter, prop string, want ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(want) == 0 {
			return nil
		}
		api, ok := testAccProvider.Meta().(*zabbix.API)
		if !ok || api == nil {
			return fmt.Errorf("testAccCheckServerAttrContains: provider not configured")
		}
		id, err := testAccStateID(s, addr)
		if err != nil {
			return err
		}
		obj, err := get(api, id)
		if err != nil {
			return fmt.Errorf("%s (%s): re-reading from the server: %s", addr, id, err)
		}
		raw, ok := serverValue(obj, prop)
		if !ok {
			return fmt.Errorf("%s (%s): the server did not return %q at all", addr, id, prop)
		}
		got := serverString(raw)
		var missing []string
		for _, w := range want {
			if !strings.Contains(got, w) {
				missing = append(missing, w)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("%s (%s): stored %s is %q, which does not contain %s",
				addr, id, prop, got, strings.Join(missing, ", "))
		}
		return nil
	}
}

// quoteHCL renders a Go string as an HCL string literal. The expressions under
// test contain double quotes and dollar signs, and both are syntax in HCL --
// "${" opens an interpolation -- so they cannot be pasted into a fixture raw.
func quoteHCL(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\t", `\t`,
		`${`, `$${`,
		`%{`, `%%{`,
	)
	return `"` + r.Replace(s) + `"`
}
