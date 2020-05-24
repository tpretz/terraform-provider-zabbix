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

def safeTfName(inname):
  return re.sub(r'[^0-9a-z]+', '-', inname, flags=re.IGNORECASE).lower()

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

        # extract groups

        # extract applications

        # extract items
        for item in tmpl.findall("items/item"):
            itemobj = {}
            for child in item:
                if child.text is None or child.text == '0':
                    continue
                itemobj[child.tag] = child.text
                log.debug("%s: %s" % (child.tag, child.text)) 
            if 'key' not in itemobj or 'value_type' not in itemobj:
                continue
            itemobj["key_safe"] = safeTfName(itemobj["key"])
            obj['items'].append(itemobj)
        log.debug("got tmpl object %s" % obj)
        templates.append(obj)
    return templates

def extractTriggers(root):
    triggers = []

    # extract triggers
    for t in root.findall("triggers/trigger"):
        tobj = {}
        for child in t:
            if child.text is None or child.text == '0':
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

def renderItem(t, i, args):
    lines = []

    ty = i.get("type", "0")
    if ty is "0": # agent
       pass
    elif ty is "1" or ty is "4" or ty is "6": # snmp
       lines.append('resource "zabbix_item_snmp" "{}" {{'.format(i['key_safe']))
       lines.append('  hostid = zabbix_template.{}.id'.format(t["template_safe"]))
       lines.append('  name = "{}"'.format(i["name"]))
       lines.append('  key = "{}"'.format(i["key"]))
       lines.append('  valuetype = "{}"'.format(ITEM_T_MAP[i["value_type"]]))
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

    print("\n".join(lines))

def renderTrigger(t):
    lines = []

    lines.append('resource "zabbix_trigger" "{}" {{'.format(t['name_safe']))
    lines.append('  name = "{}"'.format(t["name"]))
    lines.append('  expression = "{}"'.format(t["expression"]))
    lines.append('  description = "{}"'.format('\\n'.join(t["description"].splitlines())))
    lines.append('  priority = "{}"'.format(TRIGGER_P_MAP[t["priority"]]))
    if t.get('recovery_mode') == "1":
        lines.append('  recovery_expression = "{}"'.format(t['recovery_expression']))
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
    print("%s" % args)

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
