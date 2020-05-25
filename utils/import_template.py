#!/usr/bin/env python3

import argparse
import logging
import xml.etree.ElementTree as ET
import re

log = None
ITEM_T_MAP = {
  "0": "float",
  "1": "character",
  "2": "log",
  "3": "unsigned",
  "4": "text",
}
TRIGGER_P_MAP = {
  "0": "not_classified",
  "1": "info",
  "2": "warn",
  "3": "average",
  "4": "high",
  "5": "disaster",
}
EVALTYPE_P_MAP = { 
  "0": "andor",
  "1": "and",
  "2": "or",
  "3": "custom",
}
OPERATOR_P_MAP = { 
  "8": "match",
  "9": "notmatch",
}

names = {}
names_rev = {}
item_cache = {}

def hasCachedTfName(inname):
    return inname in names

def safeTfName(inname):
    if inname in names:
        return names[inname]
    gen = re.sub(r'[^0-9a-z]+', '-', inname, flags=re.IGNORECASE).lower().strip('-')
    lookup = gen

    i = 0
    while lookup in names_rev:
        lookup = gen + '-' + str(i)
        i += 1
    names[inname] = lookup
    names_rev[lookup] = inname

    return lookup

def expressionRef(ex):
    def subTmplFun(match):
        cap = match.group(1)
        log.debug("evaluating %s" % cap)

        # check if a ref, local cache of refs
        if hasCachedTfName(cap):
            return '{${zabbix_template.'+safeTfName(cap)+'.host}:'

        # will be used once conditional in place
        return '{'+cap+':'
    
    def subItemFun(match):
        cap = match.group(1)
        log.debug("evaluating %s" % cap)

        # check if a ref, local cache of refs
        if hasCachedTfName(cap):
            safe = safeTfName(cap)
            if cap in item_cache:
                return ':${'+item_cache[cap]['resource_type']+'.'+safeTfName(cap)+'.key}.'+match.group(2)+'('

        # will be used once conditional in place
        return ':'+cap+'.'

    res = re.sub(r'{([^{:]+):',
    subTmplFun,
    ex)

    res = re.sub(r':(.+)\.(\w+)\(',
    subItemFun,
    res)

    return res

def extractTemplates(root):
    templates = []

    for tmpl in root.findall("templates/template"):
        log.debug("got tmpl xml: %s" % tmpl)
    
        obj = {
            'template': getattr(tmpl.find('template'), 'text', None),
            'name': getattr(tmpl.find('name'), 'text', None),
            'description': getattr(tmpl.find('description'), 'text', None),
            'groups': [],
            'applications': [],
            'items': [],
            'triggers': [],
        }
        obj["template_safe"] = safeTfName(obj["template"])

        # extract discovery rules
        obj["rules"] = extractLLDRules(tmpl)
        log.debug("got rules %s" % obj['rules'])

        # extract groups

        # extract applications

        # extract items
        for item in tmpl.findall("items/item"):
            itemobj = {}
            for child in item:
                if child.text is None:
                    continue
                itemobj[child.tag] = child.text
                log.debug("%s: %s" % (child.tag, child.text)) 
            if 'key' not in itemobj or 'value_type' not in itemobj:
                continue
            itemobj["key_safe"] = safeTfName(itemobj["key"])
            obj['items'].append(itemobj)
            item_cache[itemobj["key"]] = itemobj

        log.debug("got tmpl object %s" % obj)
        templates.append(obj)
    return templates

def extractLLDRules(root):
    rules = []

    for t in root.findall("discovery_rules/discovery_rule"):
        tobj = {}
        for child in t:
            if child.text is None:
                continue
            if len(child) > 0:
                continue
            tobj[child.tag] = child.text
            log.debug("lld attr %s: %s" % (child.tag, child.text)) 

        tobj["name_safe"] = safeTfName(tobj["name"])

        items = []
        # nested things too
        for i in t.findall('item_prototypes/item_prototype'):
            iobj = {}
            for child in i:
                if child.text is None:
                    continue
                if len(child) > 0:
                    continue
                iobj[child.tag] = child.text
                log.debug("lld item attr %s: %s" % (child.tag, child.text)) 

            iobj["key_safe"] = safeTfName(iobj["key"])
            iobj["prototype"] = True
            item_cache[iobj["key"]] = iobj
            items.append(iobj)

        tobj['items'] = items
        
        
        triggers = []
        # nested things too
        for i in t.findall('trigger_prototypes/trigger_prototype'):
            iobj = {}
            for child in i:
                if child.text is None:
                    continue
                if len(child) > 0:
                    continue
                iobj[child.tag] = child.text
                log.debug("lld trigger attr %s: %s" % (child.tag, child.text)) 

            iobj["name_safe"] = safeTfName(iobj["name"])
            triggers.append(iobj)

        tobj['triggers'] = triggers

        rules.append(tobj)
    return rules

def extractTriggers(root):
    triggers = []

    # extract triggers
    for t in root.findall("triggers/trigger"):
        tobj = {}
        for child in t:
            if child.text is None:
                continue
            tobj[child.tag] = child.text
            log.debug("%s: %s" % (child.tag, child.text)) 
        tobj['name_safe'] = safeTfName(tobj['name'])
        triggers.append(tobj)

    log.debug("got triggers %s" % triggers)
    return triggers

def renderTemplate(t, args):
    if t.get("name") is None:
      t["name"] = t["template"]
    if t.get("description") is None:
      t["description"] = "Imported Template"
    print("""
resource "zabbix_template" "{template_safe}" {{
  host = "{template}"
  name = "{name}"

  description = "{description}"
  groups = [ "1" ]
}}
""".format_map(t))


    # now for all items
    for i in t["items"]:
      renderItem(t, i, args)

    # for all discovery rules
    for r in t["rules"]:
        renderLLDRule(t, r, args)

def renderLLDRule(t, i, args):
    lines = []
    common_lines = [
        '  hostid = zabbix_template.{}.id'.format(t["template_safe"]),
        '  name = "{}"'.format(i["name"]),
        '  key = "{}"'.format(i["key"]),
    ]
    ty = i.get("type", "0")
    if ty is "1" or ty is "4" or ty is "6": # snmp
        t['resource_type'] = 'zabbix_lld_snmp'
        lines.append('resource "{}" "{}" {{'.format(t['resource_type'], i['name_safe']))
        lines.extend(common_lines)
        lines.append('  snmp_oid = "{}"'.format(i["snmp_oid"]))
        lines.append('  snmp_version = "{}"'.format(args.snmp))
        lines.append('}')
    else:
        log.debug("unsupported discovery type")
    
    print("\n".join(lines))

    # process its items too
    for item in i['items']:
        renderLLDItem(t, i, item, args)
    
    # process its triggers too
    for trigger in i['triggers']:
        renderLLDTrigger(t, i, trigger, args)

def renderLLDItem(t, lld, i, args):
    log.info("got lld item %s" % i)
    lines = []
    common_lines = [
       '  hostid = zabbix_template.{}.id'.format(t["template_safe"]),
       '  ruleid = {}.{}.id'.format(t["resource_type"], lld["name_safe"]),
       '  name = "{}"'.format(i["name"]),
       '  key = "{}"'.format(i["key"]),
       '  valuetype = "{}"'.format(ITEM_T_MAP[i["value_type"]]),
    ]

    ty = i.get("type", "0")
    if ty is "0": # agent
       pass
    elif ty is "1" or ty is "4" or ty is "6": # snmp
        i['resource_type'] = 'zabbix_proto_item_snmp'
        lines.append('resource "{}" "{}" {{'.format(i['resource_type'], i['key_safe']))
        lines.extend(common_lines)
        lines.append('  snmp_oid = "{}"'.format(i["snmp_oid"]))
        lines.append('  snmp_version = "{}"'.format(args.snmp))
        lines.append('}')
    elif ty is "2": # trapper
       pass
    elif ty is "3": # simple
       pass
    elif ty is "5": # internal
       pass
    elif ty is "7": # active agent
       pass
    elif ty is "8": # aggregate
        i['resource_type'] = 'zabbix_proto_item_aggregate'
        lines.append('resource "{}" "{}" {{'.format(i['resource_type'], i['key_safe']))
        lines.extend(common_lines)
        lines.append('}')
    elif ty is "10": # external
       pass
    elif ty == "15": # calculated
        i['resource_type'] = 'zabbix_proto_item_calculated'
        lines.append('resource "{}" "{}" {{'.format(i['resource_type'], i['key_safe']))
        lines.extend(common_lines)
        lines.append('  formula = "{}"'.format(i['params']))
        lines.append('}')

    print("\n".join(lines))

def renderLLDTrigger(tmpl, lld, t, args):
    log.info("got trigger %s" % t)

    try:
        expression = expressionRef(t["expression"])
    except KeyError:
        log.error("cant render trigger %s as no valid item found" % t['name'])
        return

    lines = []

    lines.append('resource "zabbix_proto_trigger" "{}" {{'.format(t['name_safe']))
    lines.append('  name = "{}"'.format(t["name"]))
    lines.append('  expression = "{}"'.format(expression))
    lines.append('  comments = "{}"'.format('\\n'.join(t["description"].splitlines())))
    lines.append('  priority = "{}"'.format(TRIGGER_P_MAP[t["priority"]]))
    if t.get('recovery_mode') == "1":
        lines.append('  recovery_expression = "{}"'.format(expressionRef(t['recovery_expression'])))
    lines.append('}')

    print("\n".join(lines))

def renderItem(t, i, args):
    lines = []
    common_lines = [
       '  hostid = zabbix_template.{}.id'.format(t["template_safe"]),
       '  name = "{}"'.format(i["name"]),
       '  key = "{}"'.format(i["key"]),
       '  valuetype = "{}"'.format(ITEM_T_MAP[i["value_type"]]),
    ]

    ty = i.get("type", "0")
    if ty is "0": # agent
       pass
    elif ty is "1" or ty is "4" or ty is "6": # snmp
        i['resource_type'] = 'zabbix_item_snmp'
        lines.append('resource "{}" "{}" {{'.format(i['resource_type'], i['key_safe']))
        lines.extend(common_lines)
        lines.append('  snmp_oid = "{}"'.format(i["snmp_oid"]))
        lines.append('  snmp_version = "{}"'.format(args.snmp))
        lines.append('}')
    elif ty is "2": # trapper
       pass
    elif ty is "3": # simple
       pass
    elif ty is "5": # internal
       pass
    elif ty is "7": # active agent
       pass
    elif ty is "8": # aggregate
       pass
    elif ty is "10": # external
       pass
    elif ty is "15": # calculated 
        i['resource_type'] = 'zabbix_item_calculated'
        lines.append('resource "{}" "{}" {{'.format(i['resource_type'], i['key_safe']))
        lines.extend(common_lines)
        lines.extend('  formula = "{}"'.format(i['params']))
        lines.append('}')

    print("\n".join(lines))

def renderTrigger(t):
    lines = []

    # need to fix expression to ref both template and items for dependencies
    lines.append('resource "zabbix_trigger" "{}" {{'.format(t['name_safe']))
    lines.append('  name = "{}"'.format(t["name"]))
    lines.append('  expression = "{}"'.format(expressionRef(t["expression"])))
    lines.append('  comments = "{}"'.format('\\n'.join(t["description"].splitlines())))
    lines.append('  priority = "{}"'.format(TRIGGER_P_MAP[t["priority"]]))
    if t.get('recovery_mode') == "1":
        lines.append('  recovery_expression = "{}"'.format(expressionRef(t['recovery_expression'])))
    lines.append('}')

    print("\n".join(lines))

def main():
    logging.basicConfig(level=logging.INFO)
    global log
    log = logging.getLogger(__name__)

    parser = argparse.ArgumentParser()
    parser.add_argument('-D', '--debug', action='store_true', default=False, help='debug logging')
    parser.add_argument('-i', '--input', required=True, help='input xml file')
    parser.add_argument('-s', '--snmp', required=True, help='snmp version')

    args = parser.parse_args()

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)
        log.debug("debug logs enabled")


    tree = ET.parse(args.input)
    root = tree.getroot()
    log.debug("got xml: %s" % root)

    templates = extractTemplates(root)
    triggers = extractTriggers(root)


    for t in templates:
      renderTemplate(t, args)

    for t in triggers:
      renderTrigger(t)


if __name__ == "__main__":
    main()
