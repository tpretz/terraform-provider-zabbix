package provider

import (
	"fmt"
	"strings"

	"github.com/tpretz/terraform-provider-zabbix/internal/zabbix"
)

// Reconstructing a trigger expression.
//
// Zabbix does not store a trigger expression as the user wrote it. Every item
// reference is replaced by a {functionid} token pointing at a row in the
// functions table, and only that substitution is made -- whitespace, quoting,
// operators, parentheses, user macros and LLD macros all survive verbatim:
//
//	written  min(/host/svc.perf,{$WIN})>{$SLOW} and last(/host/svc.other)=0
//	stored   {49044}>{$SLOW} and {49045}=0
//
// trigger.get can undo that substitution itself, via "expandExpression", and
// that is what this provider used to ask for. The flag is unusable, because it
// expands user macros to their *values* at the same time and offers no way to
// separate the two:
//
//	expanded min(/host/svc.perf,5m)>5 and last(/host/svc.other)=0
//
// The macro is destroyed, so the value written to state can never equal the
// configuration, and every plan proposes rewriting the expression to its
// rendered form -- forever. See CHANGELOG 2.0.0, "trigger expressions
// containing user macros".
//
// So the substitution is undone here instead, from the raw expression plus the
// functions, items and hosts that trigger.get returns alongside it, which is
// what the Zabbix frontend does. Each function carries the item it reads and a
// parameter list whose first parameter is the placeholder "$":
//
//	functions  49044 -> itemid 75077, function "min",  parameter `$,{$WIN}`
//	           49045 -> itemid 75078, function "last", parameter `$`
//	items      75077 -> key_ "svc.perf", hostid 14487
//	           75078 -> key_ "svc.other", hostid 14487
//	hosts      14487 -> "host"
//
// Substituting "/host/key" for that "$" and wrapping the result in the function
// name rebuilds the source form exactly.
//
// Measured on 6.0.48, 7.0.29, 7.4.13 and 8.0-trunk, identically on all four:
//
//   - Functions that read no item -- date(), time(), now() -- get no functionid
//     and appear in the stored expression verbatim, so they need no handling.
//   - The parameter keeps whatever whitespace surrounded the item reference:
//     `last( /host/key )` stores the parameter as `" $ "`. The "$" is replaced
//     in place rather than the parameter rebuilt, so that spacing survives.
//   - Commas, quotes and braces inside a parameter are never interpreted --
//     `count(/h/k,1h,,"error,x")` stores `$,1h,,"error,x"` -- so the parameter
//     is never split.
//   - selectFunctions, selectItems and selectHosts cover the recovery
//     expression as well as the problem expression, in one call.
//   - A quoted string literal in an expression may contain something shaped
//     like a functionid (`last(/h/k)="{12345}"`), so string literals are copied
//     over untouched instead of being scanned.
//
// Only "$" as the *first* parameter is understood, which is where every Zabbix
// history function takes its item reference. Anything else is reported as an
// error rather than guessed at: a wrong expression silently written to state is
// the defect this code exists to fix.

// triggerExpressionResolver rebuilds source expressions for one trigger. Build
// it from a trigger read with selectFunctions/selectItems/selectHosts and use
// it for both the problem and the recovery expression.
type triggerExpressionResolver struct {
	functions map[string]zabbix.TriggerFunction
	items     map[string]zabbix.Item
	hosts     map[string]zabbix.Host
}

func newTriggerExpressionResolver(t *zabbix.Trigger) *triggerExpressionResolver {
	r := &triggerExpressionResolver{
		functions: make(map[string]zabbix.TriggerFunction, len(t.Functions)),
		items:     make(map[string]zabbix.Item, len(t.ContainedItems)),
		hosts:     make(map[string]zabbix.Host, len(t.ParentHosts)),
	}
	for _, f := range t.Functions {
		r.functions[f.FunctionID] = f
	}
	// item -> host is mapped through the item's own hostid, never through the
	// trigger's host list, which carries more than one entry as soon as the
	// expression spans hosts and says nothing about which item lives where.
	for _, i := range t.ContainedItems {
		r.items[i.ItemID] = i
	}
	for _, h := range t.ParentHosts {
		r.hosts[h.HostID] = h
	}
	return r
}

// source turns a stored expression back into the form the user wrote.
func (r *triggerExpressionResolver) source(expr string) (string, error) {
	var b strings.Builder
	b.Grow(len(expr))

	for i := 0; i < len(expr); {
		switch expr[i] {
		case '"':
			// a string literal is user data: copy it across without looking
			// inside, so that a {12345} within one is not mistaken for a
			// functionid. \" and \\ escape within.
			j := i + 1
			for j < len(expr) {
				if expr[j] == '\\' && j+1 < len(expr) {
					j += 2
					continue
				}
				j++
				if expr[j-1] == '"' {
					break
				}
			}
			b.WriteString(expr[i:j])
			i = j
		case '{':
			// {123} is a functionid; {$MACRO}, {#LLD} and {TRIGGER.VALUE} are
			// not, and are left alone.
			j := i + 1
			for j < len(expr) && expr[j] >= '0' && expr[j] <= '9' {
				j++
			}
			if j > i+1 && j < len(expr) && expr[j] == '}' {
				ref, err := r.resolve(expr[i+1 : j])
				if err != nil {
					return "", err
				}
				b.WriteString(ref)
				i = j + 1
				continue
			}
			b.WriteByte(expr[i])
			i++
		default:
			b.WriteByte(expr[i])
			i++
		}
	}

	return b.String(), nil
}

// resolve rebuilds one function call from its functionid.
func (r *triggerExpressionResolver) resolve(functionID string) (string, error) {
	fn, ok := r.functions[functionID]
	if !ok {
		return "", fmt.Errorf("trigger expression references function %s, which trigger.get did not return; the read needs selectFunctions", functionID)
	}
	if fn.Function == "" {
		// selectFunctions with an explicit field list silently drops
		// "function" on every supported version, so this is worth naming
		// rather than emitting "(...)" and letting the user work it out.
		return "", fmt.Errorf("function %s came back with no name; selectFunctions must be \"extend\"", functionID)
	}
	item, ok := r.items[fn.ItemID]
	if !ok {
		return "", fmt.Errorf("function %s reads item %s, which trigger.get did not return; the read needs selectItems", functionID, fn.ItemID)
	}
	host, ok := r.hosts[item.HostID]
	if !ok {
		return "", fmt.Errorf("item %s belongs to host %s, which trigger.get did not return; the read needs selectHosts", fn.ItemID, item.HostID)
	}

	at := itemReferenceIndex(fn.Parameter)
	if at < 0 {
		return "", fmt.Errorf("function %s (%s) has parameters %q, which do not start with the item placeholder \"$\"; the expression cannot be reconstructed", functionID, fn.Function, fn.Parameter)
	}

	var b strings.Builder
	b.WriteString(fn.Function)
	b.WriteByte('(')
	b.WriteString(fn.Parameter[:at])
	b.WriteByte('/')
	b.WriteString(host.Host)
	b.WriteByte('/')
	b.WriteString(item.Key)
	b.WriteString(fn.Parameter[at+1:])
	b.WriteByte(')')
	return b.String(), nil
}

// itemReferenceIndex locates the "$" placeholder that stands for the function's
// item reference, or -1 if the parameter list does not open with one. Leading
// whitespace is skipped rather than trimmed: `last( /host/key )` is stored as
// `" $ "`, and the spaces have to come back.
func itemReferenceIndex(parameter string) int {
	for i := 0; i < len(parameter); i++ {
		switch parameter[i] {
		case '$':
			return i
		case ' ', '\t', '\n', '\r':
		default:
			return -1
		}
	}
	return -1
}
