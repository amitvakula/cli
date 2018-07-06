import sys
import textwrap

from ..exchange import GearExchangeDB, GearVersionKey, gear_short_str, get_upgradeable_gears

def add_command(subparsers):
    parser = subparsers.add_parser('search', help='Search gear manifests')
    parser.add_argument('query', help='The string to search for')

    parser.set_defaults(func=search_gears)
    parser.set_defaults(parser=parser)

    return parser

SEARCH_KEYS = ['name', 'label', 'description', 'author', 'maintainer']
def search_gear(gear_doc, query):
    gear = gear_doc['gear']
    for key in SEARCH_KEYS:
        text = gear.get(key, '').lower()
        if query in text:
            return True
    return False

def search_gears(args):
    db = GearExchangeDB()
    db.update()

    matches = []
    # Dead simple search, could be better
    query = args.query.lower()
    for gear in db.get_latest_gears():
        if search_gear(gear, query):
            matches.append(gear)

    if not matches:
        print('Did not find any matching gears')
        sys.exit(1)

    for gear in matches:
        print(gear_short_str(gear))
        if 'description' in gear['gear']:
            # Col-width: 80 TODO: Someday we should configure this
            desc_str = '\n'.join(textwrap.wrap(gear['gear']['description'], width=78))
            print(textwrap.indent(desc_str, '  '))
            
