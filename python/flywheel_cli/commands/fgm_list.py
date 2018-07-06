import sys

from ..sdk_impl import create_flywheel_client
from ..exchange import GearExchangeDB, GearVersionKey, gear_short_str, get_upgradeable_gears

def add_command(subparsers):
    parser = subparsers.add_parser('list', help='List gears, with a few options to filter')

    group = parser.add_mutually_exclusive_group()
    group.add_argument('-i', '--installed', action='store_true', help='Show only installed gears')
    group.add_argument('-u', '--upgradeable', action='store_true', help='Show only upgradeable gears')
    group.add_argument('--group', help='Show only gears on the exchange that belong to a particular group')
    group.add_argument('--all-versions', metavar='name', help='Show all versions for the named gear')

    parser.set_defaults(func=list_gears)
    parser.set_defaults(parser=parser)

    return parser

def list_gears(args):
    db = GearExchangeDB()
    db.update()

    fw = create_flywheel_client(root=True)

    gears = []
    if args.upgradeable:
        gears = get_upgradeable_gears(db, fw) 
    elif args.installed:
        gears = fw.get_all_gears()
        gears.sort(key=lambda g: g['gear']['name'])
    elif args.group:
        gears = db.get_latest_gears(args.group)
    elif args.all_versions:
        gears = db.find_by_name(args.all_versions)
        gears.sort(key=GearVersionKey)
    else:
        gears = db.get_latest_gears()

    for gear in gears:
        print(gear_short_str(gear))
