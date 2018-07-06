import fs
import re
import json 
import os
import sys

from ..exchange import GearExchangeDB, GearVersionKey, prepare_manifest_for_upload
from ..sdk_impl import create_flywheel_client, get_site_name
from .. import util

def add_command(subparsers):
    parser = subparsers.add_parser('upgrade', help='Upgrade installed flywheel gears')
    parser.add_argument('name', nargs='*', help='The optional name of the gear to upgrade')

    parser.set_defaults(func=upgrade_gears)
    parser.set_defaults(parser=parser)

    return parser

def upgrade_gears(args):
    db = GearExchangeDB()
    db.update()

    fw = create_flywheel_client(root=True)

    # Get the list of installed gears (possibly filtering by name)
    installed_gear_map = {}
    for gear in fw.get_all_gears():
        name = gear.gear.name
        if not args.name or name in args.name:
            if name in installed_gear_map:
                if gear.created < installed_gear_map[name].created:
                    print('Ignoring older version of {}'.format(name))
                    continue

            installed_gear_map[name] = gear

    # The list of candidates for upgrades
    upgrades = []
    for gear_name in sorted(installed_gear_map.keys()):
        gear = installed_gear_map[gear_name]

        current_version = GearVersionKey(gear)
        gear_doc = db.find_latest(gear_name)
        if gear_doc:
            latest_version = GearVersionKey(gear_doc)
            if latest_version > current_version:
                print('  upgrade {} from {} to {}'.format(gear_name, current_version, latest_version))
                upgrades.append((gear_name, gear.category, gear_doc))

    if not upgrades:
        # Didn't find any upgrades to perform
        if args.name and not installed_gear_map:
            print('Could not find any installed gears by that name', file=sys.stderr)
            sys.exit(1)

        print('Nothing to do!')
        sys.exit(0)

    site_name = get_site_name(fw)
    if util.confirmation_prompt('Upgrade {} gears on {}?'.format(len(upgrades), site_name)):
        for gear_name, category, gear in upgrades:
            print('Upgrading {} to {}...'.format(gear_name, gear['gear']['version']))
            prepare_manifest_for_upload(gear, category=category)
            fw.add_gear(gear_name, gear)

        print('Done!')
    else:
        print('OK then.')

