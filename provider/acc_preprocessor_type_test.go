package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Acceptance coverage for `preprocessor.type` as a named enum.
//
// schema_enum_test.go proves the validator accepts every name it advertises;
// that is a statement about the provider only. These tests prove the *server*
// accepts them, which is the part no amount of unit testing can establish: the
// list of preprocessing steps differs by Zabbix version and differs again
// between an item and a discovery rule, and the provider's tables are a claim
// about both that is only worth what it has been checked against.
//
// Every name in PREPROC_LOOKUP_ARR and LLD_PREPROC_LOOKUP_ARR is created on
// each server in the matrix, with parameters that step actually requires. A
// name added to a lookup without a fixture here fails
// TestAccPreprocessorTypeFixturesAreComplete rather than quietly going
// untested.

// preprocCase is a step type together with parameters Zabbix will accept for
// it. The parameters are not incidental: most step types reject an empty
// params list, several require an exact count, and a handful validate the
// contents, so "create an item with this type" is only a meaningful test if
// the rest of the step is right.
type preprocCase struct {
	// params sent on every version.
	params []string
	// errorHandler overrides the "0" default. check_unsupported is the only
	// step type Zabbix refuses to pair with error handler 0, on every version.
	errorHandler string
}

// preprocCases covers every name in PREPROC_LOOKUP. The parameter values were
// arrived at by creating each step against live 6.0.48, 7.0.29, 7.4.13 and 8.0
// servers and reading the rejections back, then re-reading each created step
// to confirm the server returns what was sent -- a step whose parameters come
// back normalised produces a permanent diff, which is a different failure from
// a step that will not create at all and one this test would otherwise miss.
var preprocCases = map[string]preprocCase{
	"multiplier":                  {params: []string{"2"}},
	"rtrim":                       {params: []string{"x"}},
	"ltrim":                       {params: []string{"x"}},
	"trim":                        {params: []string{"x"}},
	"regex":                       {params: []string{"(.*)", "\\1"}},
	"bool_to_decimal":             {},
	"octal_to_decimal":            {},
	"hex_to_decimal":              {},
	"simple_change":               {},
	"change_per_second":           {},
	"xml_xpath":                   {params: []string{"/a/b"}},
	"jsonpath":                    {params: []string{"$.a"}},
	"in_range":                    {params: []string{"0", "100"}},
	"matches_regex":               {params: []string{"^a"}},
	"not_matches_regex":           {params: []string{"^a"}},
	"check_json_error":            {params: []string{"$.error"}},
	"check_xml_error":             {params: []string{"/error"}},
	"check_regex_error":           {params: []string{"err:(.*)", "\\1"}},
	"discard_unchanged":           {},
	"discard_unchanged_heartbeat": {params: []string{"1h"}},
	"javascript":                  {params: []string{"return value;"}},
	// the third parameter is empty and is not optional: Zabbix stores
	// "<pattern>\nvalue\n" and returns three parameters whatever was sent.
	// See TestAccResourceItemPreprocessorEmptyParam.
	"prometheus_pattern": {params: []string{"cpu_usage_system", "value", ""}},
	"prometheus_to_json": {},
	"csv_to_json":        {params: []string{",", "\"", "1"}},
	"replace":            {params: []string{"a", "b"}},
	// deliberately no params, on every version. 6.0 rejects any parameter
	// here with `should be empty` and 7.0 requires one, and the client
	// papers over the difference itself: prepPreprocessors injects "-1"
	// ("match any error") on 7.0+ and readPreprocessors strips it back off,
	// so the version-independent configuration is the empty one. Writing
	// "-1" out in full is the one thing that does not round-trip -- the read
	// path removes it and the plan never empties -- which is what this
	// fixture asserted until a 7.0 run said otherwise.
	//
	// Error handler 0, "report the error", is meaningless for a step whose
	// whole job is to handle an error, and every version rejects it.
	"check_unsupported": {errorHandler: "1"},
	"xml_to_json":       {},
	"snmp_walk_value":   {params: []string{"1.3.6.1.2.1.1.1", "0"}},
	"snmp_walk_to_json": {params: []string{"{#IFNAME}", "1.3.6.1.2.1.2.2.1.2", "0"}},
	// the format parameter is one of 1, 2, 3 -- not 0, which the item and
	// walk steps do accept. Zabbix 7.0 through 8.0 all reject "0" here.
	"snmp_get_value": {params: []string{"1"}},
}

// hcl renders a case's params and error handler as configuration.
func (c preprocCase) hcl() string {
	params := c.params

	var b strings.Builder
	if len(params) > 0 {
		quoted := make([]string, len(params))
		for i, p := range params {
			quoted[i] = fmt.Sprintf("%q", p)
		}
		fmt.Fprintf(&b, "\t\tparams = [%s]\n", strings.Join(quoted, ", "))
	}
	if c.errorHandler != "" {
		fmt.Fprintf(&b, "\t\terror_handler = %q\n", c.errorHandler)
	}
	return b.String()
}

// TestAccPreprocessorTypeFixturesAreComplete is the guard that keeps the two
// tests below honest. Without it, adding a step type to PREPROC_LOOKUP and
// forgetting the fixture would leave the new type advertised in the schema,
// documented, and never once sent to a server.
//
// It needs no server and is not an acceptance test in the TF_ACC sense; it
// lives here so that it sits beside the thing it is guarding.
func TestAccPreprocessorTypeFixturesAreComplete(t *testing.T) {
	for _, name := range PREPROC_LOOKUP_ARR {
		if _, ok := preprocCases[name]; !ok {
			t.Errorf("no fixture for preprocessing type %q; it would never be created against a server", name)
		}
	}
	for name := range preprocCases {
		if _, ok := PREPROC_LOOKUP[name]; !ok {
			t.Errorf("fixture for %q, which is not a preprocessing type", name)
		}
	}
	// the discovery-rule list is a subset, so it needs no fixtures of its own
	for _, name := range LLD_PREPROC_LOOKUP_ARR {
		if _, ok := preprocCases[name]; !ok {
			t.Errorf("no fixture for discovery-rule preprocessing type %q", name)
		}
	}
}

const preprocTemplateHCL = `
resource "zabbix_templategroup" "testgrp" {
	name = "test-group"
}
resource "zabbix_template" "testtmpl" {
	groups = [ zabbix_templategroup.testgrp.id ]
	host = "test-template"
}
`

func preprocItemHCL(step string) string {
	return preprocTemplateHCL + `
resource "zabbix_item_trapper" "testitem" {
	hostid    = zabbix_template.testtmpl.id
	key       = "preproc.type.item"
	name      = "Preprocessing Type"
	valuetype = "float"

	preprocessor {
` + step + `	}
}
`
}

func preprocLLDHCL(step string) string {
	return preprocTemplateHCL + `
resource "zabbix_lld_trapper" "testlld" {
	hostid = zabbix_template.testtmpl.id
	key    = "preproc.type.lld"
	name   = "Preprocessing Type LLD"
	delay  = "0"

	preprocessor {
` + step + `	}
}
`
}

// TestAccPreprocessorTypeEveryItemType creates an item with every step type
// the schema offers, one step per test step, against whatever server is under
// test.
//
// The named form is what is written, and each step asserts the name survives
// the round trip -- Zabbix returns the numeric code and the read path has to
// turn it back, so a missing _REV entry would show up here as an attribute
// that reads back as a number or as nothing at all. terraform-plugin-testing
// requires an empty plan after every apply, so each step is also a
// round-trip assertion on the parameters, which is where the version
// differences actually bite.
func TestAccPreprocessorTypeEveryItemType(t *testing.T) {
	version := testAccVersion(t)
	steps := make([]resource.TestStep, 0, len(PREPROC_LOOKUP_ARR))

	for _, name := range PREPROC_LOOKUP_ARR {
		if g, gated := PREPROC_MIN_VERSION[name]; gated && version < g.version {
			continue // covered from the other side by TestAccPreprocessorTypeGated
		}
		c := preprocCases[name]
		steps = append(steps, resource.TestStep{
			Config: hcl(t, preprocItemHCL(fmt.Sprintf("\t\ttype = %q\n", name)+c.hcl())),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "preprocessor.#", "1"),
				resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "preprocessor.0.type", name),
				testAccCheckItemPreprocessorCount("zabbix_item_trapper.testitem", 1),
			),
		})
	}

	if len(steps) == 0 {
		t.Fatal("no step types to test; PREPROC_LOOKUP_ARR is empty or every type is gated")
	}
	t.Logf("server %s: %d of %d item preprocessing types", zabbixVersionString(version), len(steps), len(PREPROC_LOOKUP_ARR))

	// import at the end, with whatever the last step left behind, so the
	// flatten function is checked against the importer as well
	steps = append(steps, resource.TestStep{
		ResourceName:      "zabbix_item_trapper.testitem",
		ImportState:       true,
		ImportStateVerify: true,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             steps,
	})
}

// TestAccPreprocessorTypeEveryLLDType is the same for a discovery rule, whose
// list is a different list. It is not a formality: the LLD path has its own
// generate and flatten functions in common_lld.go and its own lookup, so
// nothing the item test does says anything about it.
func TestAccPreprocessorTypeEveryLLDType(t *testing.T) {
	version := testAccVersion(t)
	steps := make([]resource.TestStep, 0, len(LLD_PREPROC_LOOKUP_ARR))

	for _, name := range LLD_PREPROC_LOOKUP_ARR {
		if g, gated := LLD_PREPROC_MIN_VERSION[name]; gated && version < g.version {
			continue
		}
		c := preprocCases[name]
		steps = append(steps, resource.TestStep{
			Config: hcl(t, preprocLLDHCL(fmt.Sprintf("\t\ttype = %q\n", name)+c.hcl())),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.#", "1"),
				resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.type", name),
				testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 1),
			),
		})
	}

	if len(steps) == 0 {
		t.Fatal("no discovery-rule step types to test")
	}
	t.Logf("server %s: %d of %d discovery-rule preprocessing types", zabbixVersionString(version), len(steps), len(LLD_PREPROC_LOOKUP_ARR))

	steps = append(steps, resource.TestStep{
		ResourceName:      "zabbix_lld_trapper.testlld",
		ImportState:       true,
		ImportStateVerify: true,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps:             steps,
	})
}

// skipAtOrAbove is skipBelow's mirror: it skips a step on a server that *has*
// the feature, for the tests that assert what happens on one that does not.
func skipAtOrAbove(t *testing.T, version int) func() (bool, error) {
	return func() (bool, error) {
		return testAccVersion(t) >= version, nil
	}
}

// TestAccPreprocessorTypeGated is the other half of the two tests above: the
// types they skip because the server is too old have to be *refused*, and
// refused with something a user can act on.
//
// The refusal has to come from the provider. Left to the server, 6.0 answers
// `Incorrect value for field "type": unexpected value "30"` -- a number the
// user never wrote, on a field whose path it does not give, for a reason it
// does not state. The schema cannot do it either: a ValidateFunc runs before
// the provider has connected to anything.
func TestAccPreprocessorTypeGated(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // snmp_walk_value arrived in 6.4
				SkipFunc: skipAtOrAbove(t, zabbix.V64),
				Config: hcl(t, preprocItemHCL(`		type = "snmp_walk_value"
		params = ["1.3.6.1.2.1.1.1", "0"]
`)),
				ExpectError: regexp.MustCompile(`"snmp_walk_value" \(Zabbix code 28\) requires Zabbix 6\.4 or later; this server is 6\.0`),
			},
			{ // snmp_get_value arrived in 7.0
				SkipFunc: skipAtOrAbove(t, zabbix.V70),
				Config: hcl(t, preprocItemHCL(`		type = "snmp_get_value"
		params = ["1"]
`)),
				ExpectError: regexp.MustCompile(`"snmp_get_value" \(Zabbix code 30\) requires Zabbix 7\.0 or later`),
			},
			{ // and the deprecated numeric form is gated identically; it is
				// resolved to a name before the gate is consulted, so a v0.x
				// configuration gets the same message rather than the server's
				SkipFunc:    skipAtOrAbove(t, zabbix.V70),
				Config:      hcl(t, preprocItemHCL("\t\ttype = \"30\"\n\t\tparams = [\"1\"]\n")),
				ExpectError: regexp.MustCompile(`"snmp_get_value" \(Zabbix code 30\) requires Zabbix 7\.0 or later`),
			},
			{ // the discovery-rule-only gate: 6.0 takes matches_regex on an
				// item and rejects it on a rule
				SkipFunc: skipAtOrAbove(t, zabbix.V70),
				Config: hcl(t, preprocLLDHCL(`		type = "matches_regex"
		params = ["^a"]
`)),
				ExpectError: regexp.MustCompile(`"matches_regex" \(Zabbix code 14\) requires Zabbix 7\.0 or later`),
			},
			{ // ... which is exactly what an item is allowed to do on the same
				// server, and the reason the two lists carry separate gates
				SkipFunc: skipAtOrAbove(t, zabbix.V70),
				Config: hcl(t, preprocItemHCL(`		type = "matches_regex"
		params = ["^a"]
`)),
				Check: resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "preprocessor.0.type", "matches_regex"),
			},
			{ // every step above is skipped on a current server, and a
				// TestCase with nothing left to run is not a pass worth
				// having; this one runs everywhere
				Config: hcl(t, preprocTemplateHCL),
				Check:  resource.TestCheckResourceAttr("zabbix_template.testtmpl", "host", "test-template"),
			},
		},
	})
}

// TestAccPreprocessorTypeNumericCompat is the compatibility contract in full.
//
// A v0.x configuration writes `type = "12"`. It must still apply, it must not
// be left in a permanent diff, and the state it produces must be the canonical
// name -- which together mean the user can upgrade the provider without
// touching the configuration, and rewrite the configuration whenever they get
// to it.
//
// The PlanOnly step is the load-bearing one. Without the StateFunc that
// rewrites "12" to "jsonpath" before the value is compared, state would hold
// the name the read path wrote and configuration would hold the number, and
// every plan from then on would show a diff that applying could not clear.
func TestAccPreprocessorTypeNumericCompat(t *testing.T) {
	itemNumeric := preprocItemHCL(`		type = "12"
		params = ["$.a"]
`)
	itemNamed := preprocItemHCL(`		type = "jsonpath"
		params = ["$.a"]
`)
	lldNumeric := preprocLLDHCL(`		type = "21"
		params = ["return value;"]
`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAllDestroyed,
		Steps: []resource.TestStep{
			{ // apply the v0.x form: it works, and state holds the name
				Config: hcl(t, itemNumeric),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_item_trapper.testitem", "preprocessor.0.type", "jsonpath"),
					testAccCheckItemPreprocessorCount("zabbix_item_trapper.testitem", 1),
				),
			},
			{ // and it has converged: the same numeric configuration now plans
				// empty against the name in state
				Config:   hcl(t, itemNumeric),
				PlanOnly: true,
			},
			{ // rewriting the configuration to the name is a no-op, which is
				// what makes the migration safe to do at leisure
				Config:   hcl(t, itemNamed),
				PlanOnly: true,
			},
			{ // import sees the name too, not the code: ImportStateVerify
				// compares the imported attributes against the ones in state,
				// and state holds "jsonpath"
				ResourceName:      "zabbix_item_trapper.testitem",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{ // the discovery-rule path has its own lookup and its own
				// StateFunc instance; neither is exercised by the item steps
				Config: hcl(t, lldNumeric),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zabbix_lld_trapper.testlld", "preprocessor.0.type", "javascript"),
					testAccCheckLLDPreprocessorCount("zabbix_lld_trapper.testlld", 1),
				),
			},
			{
				Config:   hcl(t, lldNumeric),
				PlanOnly: true,
			},
		},
	})
}
