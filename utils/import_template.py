#!/usr/bin/env python3

import argparse
import logging
import xml.etree.ElementTree as ET

def main():
    logging.basicConfig(level=logging.INFO)
    log = logging.getLogger(__name__)

    parser = argparse.ArgumentParser()
    parser.add_argument('-D', '--debug', action='store_true', default=False, help='debug logging')
    parser.add_argument('-i', '--input', required=True, help='input xml file')

    args = parser.parse_args()
    print("%s" % args)

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)
        log.debug("debug logs enabled")


    tree = ET.parse(args.input)
    root = tree.getroot()
    log.debug("got xml: %s" % root)

    triggers = []

    for tmpl in root.findall("templates/template"):
        log.debug("got tmpl xml: %s" % tmpl)
    
        obj = {
            'host': getattr(tmpl.find('template'), 'text', None),
            'name': getattr(tmpl.find('name'), 'text', None),
            'description': getattr(tmpl.find('description'), 'text', None),
            'none': getattr(tmpl.find('none'), 'text', None),
            'groups': [],
            'applications': [],
            'items': [],
            'triggers': [],
        }

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
            obj['items'].append(itemobj)
        log.debug("got tmpl object %s" % obj)

    # extract triggers
    for t in root.findall("triggers/trigger"):
        tobj = {}
        for child in t:
            if child.text is None or child.text == '0':
                continue
            tobj[child.tag] = child.text
            log.debug("%s: %s" % (child.tag, child.text)) 
        triggers.append(tobj)

    log.debug("got triggers %s" % triggers)
    

if __name__ == "__main__":
    main()
