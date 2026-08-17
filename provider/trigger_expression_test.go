package provider

import (
	"strings"
	"testing"

	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// The fixtures below are transcribed from trigger.get responses captured
// against live 6.0.48, 7.0.29, 7.4.13 and 8.0-trunk servers. Every one of the
// four returned byte-identical functions/items/hosts for the same expression,
// so one table covers all of them; the acceptance tests in
// acc_update_trigger_test.go are what re-check that against real servers.
//
// Shorthand, so the table stays readable:
//
//	function  "functionid|itemid|name|parameter"
//	item      "itemid|hostid|key_"
//	host      "hostid|host"

func triggerFixture(fns, items, hosts []string) *zabbix.Trigger {
	t := &zabbix.Trigger{}
	for _, s := range fns {
		p := strings.SplitN(s, "|", 4)
		t.Functions = append(t.Functions, zabbix.TriggerFunction{
			FunctionID: p[0], ItemID: p[1], Function: p[2], Parameter: p[3],
		})
	}
	for _, s := range items {
		p := strings.SplitN(s, "|", 3)
		t.ContainedItems = append(t.ContainedItems, zabbix.Item{
			ItemID: p[0], HostID: p[1], Key: p[2],
		})
	}
	for _, s := range hosts {
		p := strings.SplitN(s, "|", 2)
		t.ParentHosts = append(t.ParentHosts, zabbix.Host{HostID: p[0], Host: p[1]})
	}
	return t
}

// the item and host inventory shared by most cases
var (
	tfHosts = []string{"10|h1", "11|h2", "12|spaced host name", "13|a-template"}
	tfItems = []string{
		"20|10|svc.perf",
		"21|10|svc.other",
		"22|10|svc.txt",
		"23|11|svc.perf",
		"24|12|net.tcp.service.perf[https,,443]",
		"25|12|vfs.fs.size[/,pused]",
		"26|13|proto[{#N}]",
		"27|13|tsvc.perf",
	}
)

func TestTriggerExpressionSource(t *testing.T) {
	tests := []struct {
		name   string
		stored string
		fns    []string
		want   string
	}{
		// -- the reported bug ------------------------------------------------
		{
			name:   "user macro on the right hand side",
			stored: "{1}>{$SLOW}",
			fns:    []string{"1|20|min|$,5m"},
			want:   "min(/h1/svc.perf,5m)>{$SLOW}",
		},
		{
			name:   "user macro inside a function parameter",
			stored: "{1}>{$SLOW}",
			fns:    []string{"1|20|min|$,{$WIN}"},
			want:   "min(/h1/svc.perf,{$WIN})>{$SLOW}",
		},
		{
			name:   "user macro with a quoted context",
			stored: "{1}>{$SLOW}",
			fns:    []string{`1|25|min|$,{$WIN:"ctx"}`},
			want:   `min(/spaced host name/vfs.fs.size[/,pused],{$WIN:"ctx"})>{$SLOW}`,
		},
		{
			name:   "user macro context containing a comma",
			stored: "{1}>0",
			fns:    []string{`1|20|min|$,{$WIN:"a,b"}`},
			want:   `min(/h1/svc.perf,{$WIN:"a,b"})>0`,
		},
		{
			// the configuration from the bug report, near enough
			name:   "host name containing spaces",
			stored: "{1}>{$HTTPS.RESPONSE.SLOW}",
			fns:    []string{"1|24|min|$,5m"},
			want:   "min(/spaced host name/net.tcp.service.perf[https,,443],5m)>{$HTTPS.RESPONSE.SLOW}",
		},

		// -- shapes with no macro in them, which must still round trip -------
		{
			name:   "no functions at all",
			stored: "1=1",
			want:   "1=1",
		},
		{
			name:   "the trivial case",
			stored: "{1}>10",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)>10",
		},
		{
			name:   "two functions",
			stored: "{1}>{$SLOW} and {2}=0",
			fns:    []string{"1|20|min|$,5m", "2|21|last|$"},
			want:   "min(/h1/svc.perf,5m)>{$SLOW} and last(/h1/svc.other)=0",
		},
		{
			name:   "the same function twice over different items",
			stored: "{1}>{2}",
			fns:    []string{"1|20|last|$", "2|21|last|$"},
			want:   "last(/h1/svc.perf)>last(/h1/svc.other)",
		},
		{
			name:   "two functions over the same item",
			stored: "{1}>0 and {2}<9",
			fns:    []string{"1|20|min|$,5m", "2|20|max|$,5m"},
			want:   "min(/h1/svc.perf,5m)>0 and max(/h1/svc.perf,5m)<9",
		},
		{
			// selectHosts alone is ambiguous here; the item's own hostid is
			// what decides which host each reference names
			name:   "spanning two hosts",
			stored: "{1}>0 and {2}>0",
			fns:    []string{"1|20|last|$", "2|23|last|$"},
			want:   "last(/h1/svc.perf)>0 and last(/h2/svc.perf)>0",
		},
		{
			name:   "functions taking no item are stored verbatim",
			stored: "{1}>0 and date()>20200101 and time()>120000 and now()>0",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)>0 and date()>20200101 and time()>120000 and now()>0",
		},
		{
			name:   "not, or and parentheses",
			stored: "not ({1}=0) or {2}=0",
			fns:    []string{"1|20|last|$", "2|21|last|$"},
			want:   "not (last(/h1/svc.perf)=0) or last(/h1/svc.other)=0",
		},
		{
			name:   "a function wrapping other functions",
			stored: "abs({1}-{2})>5",
			fns:    []string{"1|20|last|$", "2|21|last|$"},
			want:   "abs(last(/h1/svc.perf)-last(/h1/svc.other))>5",
		},
		{
			name:   "arithmetic on a function result",
			stored: "{1}*8>{$SLOW}",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)*8>{$SLOW}",
		},
		{
			name:   "time shift and count specifiers",
			stored: "{1}>{2}",
			fns:    []string{"1|20|last|$,#1", "2|20|avg|$,1h:now-1d"},
			want:   "last(/h1/svc.perf,#1)>avg(/h1/svc.perf,1h:now-1d)",
		},
		{
			name:   "an item key containing a slash and a comma",
			stored: "{1}>{$SLOW}",
			fns:    []string{"1|25|last|$"},
			want:   "last(/spaced host name/vfs.fs.size[/,pused])>{$SLOW}",
		},
		{
			name:   "a trigger on a template",
			stored: "{1}>{$SLOW}",
			fns:    []string{"1|27|min|$,{$WIN}"},
			want:   "min(/a-template/tsvc.perf,{$WIN})>{$SLOW}",
		},

		// -- parameters are never split --------------------------------------
		{
			name:   "an empty parameter and a quoted comma",
			stored: "{1}>0",
			fns:    []string{`1|22|count|$,1h,,"error,x"`},
			want:   `count(/h1/svc.txt,1h,,"error,x")>0`,
		},
		{
			name:   "several quoted parameters",
			stored: "{1}>0",
			fns:    []string{`1|22|count|$,1h,"like","err"`},
			want:   `count(/h1/svc.txt,1h,"like","err")>0`,
		},
		{
			name:   "a regular expression parameter",
			stored: "{1}=1",
			fns:    []string{`1|22|find|$,,"regexp","^err.*$"`},
			want:   `find(/h1/svc.txt,,"regexp","^err.*$")=1`,
		},
		{
			name:   "an escaped quote inside a parameter",
			stored: "{1}>0",
			fns:    []string{`1|22|count|$,1h,,"a\"b"`},
			want:   `count(/h1/svc.txt,1h,,"a\"b")>0`,
		},
		{
			name:   "a literal dollar in a later parameter",
			stored: "{1}>0",
			fns:    []string{`1|22|count|$,1h,,"$"`},
			want:   `count(/h1/svc.txt,1h,,"$")>0`,
		},
		{
			name:   "something shaped like a functionid inside a parameter",
			stored: "{1}>0",
			fns:    []string{`1|22|count|$,1h,,"{12345}"`},
			want:   `count(/h1/svc.txt,1h,,"{12345}")>0`,
		},

		// -- whitespace and literals in the expression body ------------------
		{
			name:   "whitespace around the item reference",
			stored: "{1}>10",
			fns:    []string{"1|20|last| $ "},
			want:   "last( /h1/svc.perf )>10",
		},
		{
			name:   "spaces around the operator",
			stored: "{1} > 10",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf) > 10",
		},
		{
			name:   "tabs around the operator",
			stored: "{1}\t>\t10",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)\t>\t10",
		},
		{
			name:   "a newline in the expression",
			stored: "{1}>10\n  and {2}>1",
			fns:    []string{"1|20|last|$", "2|21|last|$"},
			want:   "last(/h1/svc.perf)>10\n  and last(/h1/svc.other)>1",
		},
		{
			name:   "a string literal shaped like a functionid",
			stored: `{1}="{12345}"`,
			fns:    []string{"1|22|last|$"},
			want:   `last(/h1/svc.txt)="{12345}"`,
		},
		{
			name:   "a string literal containing a real functionid",
			stored: `{1}="a{1}b"`,
			fns:    []string{"1|22|last|$"},
			want:   `last(/h1/svc.txt)="a{1}b"`,
		},
		{
			name:   "escaped quotes in a string literal",
			stored: `{1}="say \"hi\""`,
			fns:    []string{"1|22|last|$"},
			want:   `last(/h1/svc.txt)="say \"hi\""`,
		},
		{
			name:   "a string literal containing a comma",
			stored: `{1}="a,b"`,
			fns:    []string{"1|22|last|$"},
			want:   `last(/h1/svc.txt)="a,b"`,
		},
		{
			name:   "an empty string literal",
			stored: `{1}=""`,
			fns:    []string{"1|22|last|$"},
			want:   `last(/h1/svc.txt)=""`,
		},

		// -- macros that are not functionids ---------------------------------
		{
			name:   "an LLD macro in the item key",
			stored: "{1}>0",
			fns:    []string{"1|26|last|$"},
			want:   "last(/a-template/proto[{#N}])>0",
		},
		{
			name:   "an LLD macro in a function parameter",
			stored: "{1}>{$SLOW}",
			fns:    []string{`1|26|count|$,1h,,"{#VAL}"`},
			want:   `count(/a-template/proto[{#N}],1h,,"{#VAL}")>{$SLOW}`,
		},
		{
			name:   "a built-in macro in the expression body",
			stored: "{1}=0 and {TRIGGER.VALUE}=1",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)=0 and {TRIGGER.VALUE}=1",
		},
		{
			name:   "an unterminated brace is left alone",
			stored: "{1}>0 and {notaclosedbrace",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)>0 and {notaclosedbrace",
		},
		{
			name:   "an empty brace pair is left alone",
			stored: "{1}>0 and {}=1",
			fns:    []string{"1|20|last|$"},
			want:   "last(/h1/svc.perf)>0 and {}=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTriggerExpressionResolver(triggerFixture(tt.fns, tfItems, tfHosts))
			got, err := r.source(tt.stored)
			if err != nil {
				t.Fatalf("source(%q) errored: %v", tt.stored, err)
			}
			if got != tt.want {
				t.Errorf("source(%q)\n got %q\nwant %q", tt.stored, got, tt.want)
			}
		})
	}
}

// The recovery expression is reconstructed from the same three collections as
// the problem expression -- trigger.get returns the functions, items and hosts
// of both in one response, which is measured behaviour on all four versions and
// not an assumption. This is the case that proves it: the recovery expression
// names an item, and a host, that the problem expression does not.
func TestTriggerExpressionSourceRecovery(t *testing.T) {
	trig := triggerFixture(
		[]string{"1|20|last|$", "2|21|last|$", "3|23|last|$"},
		tfItems, tfHosts,
	)
	r := newTriggerExpressionResolver(trig)

	got, err := r.source("{1}>{$SLOW}")
	if err != nil {
		t.Fatalf("problem expression errored: %v", err)
	}
	if want := "last(/h1/svc.perf)>{$SLOW}"; got != want {
		t.Errorf("problem expression\n got %q\nwant %q", got, want)
	}

	got, err = r.source("{2}<1 and {3}<1")
	if err != nil {
		t.Fatalf("recovery expression errored: %v", err)
	}
	if want := "last(/h1/svc.other)<1 and last(/h2/svc.perf)<1"; got != want {
		t.Errorf("recovery expression\n got %q\nwant %q", got, want)
	}
}

// Nothing here should ever happen against a healthy server. Each is a way the
// reconstruction could go quietly wrong, and each is reported instead: writing
// a wrong expression into state without saying so is the defect this code was
// added to fix, so it must not be the failure mode of the fix.
func TestTriggerExpressionSourceErrors(t *testing.T) {
	tests := []struct {
		name    string
		stored  string
		fns     []string
		items   []string
		hosts   []string
		wantErr string
	}{
		{
			name:    "functionid not returned by the server",
			stored:  "{99}>0",
			fns:     []string{"1|20|last|$"},
			items:   tfItems,
			hosts:   tfHosts,
			wantErr: "selectFunctions",
		},
		{
			name:    "function with no name",
			stored:  "{1}>0",
			fns:     []string{"1|20||$"},
			items:   tfItems,
			hosts:   tfHosts,
			wantErr: "no name",
		},
		{
			name:    "item not returned by the server",
			stored:  "{1}>0",
			fns:     []string{"1|99|last|$"},
			items:   tfItems,
			hosts:   tfHosts,
			wantErr: "selectItems",
		},
		{
			name:    "host not returned by the server",
			stored:  "{1}>0",
			fns:     []string{"1|20|last|$"},
			items:   []string{"20|99|svc.perf"},
			hosts:   tfHosts,
			wantErr: "selectHosts",
		},
		{
			name:    "parameters that do not open with the placeholder",
			stored:  "{1}>0",
			fns:     []string{"1|20|last|5m,$"},
			items:   tfItems,
			hosts:   tfHosts,
			wantErr: "item placeholder",
		},
		{
			name:    "empty parameters",
			stored:  "{1}>0",
			fns:     []string{"1|20|last|"},
			items:   tfItems,
			hosts:   tfHosts,
			wantErr: "item placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTriggerExpressionResolver(triggerFixture(tt.fns, tt.items, tt.hosts))
			got, err := r.source(tt.stored)
			if err == nil {
				t.Fatalf("source(%q) returned %q, expected an error", tt.stored, got)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}
