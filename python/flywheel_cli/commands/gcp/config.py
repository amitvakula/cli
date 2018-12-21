import argparse
import collections
import json
import os
import sys

from ...errors import CliError
from ...sdk_impl import create_flywheel_session
from ...util import set_subparser_print_help


CONFIG_PATH = '~/.config/flywheel/gcp.json'

PROPERTIES = collections.OrderedDict({
    'project': 'Google Cloud Platform project',
    'hcapi.location': 'Healthcare API location (region)',
    'hcapi.dataset': 'Healthcare API dataset',
    'hcapi.dicomstore': 'Healthcare API dicom-store',
})

SET_PARSER_DESC = """
example:
  fw gcp config set project=my-project hcapi.location=us-central1

available config properties:
""" + '\n'.join('  {:16.16} {}'.format(key, desc) for key, desc in PROPERTIES.items())


class Config(dict):
    def __init__(self):
        self.path = os.path.expanduser(CONFIG_PATH)
        try:
            with open(self.path, 'r') as f:
                config = json.load(f)
        except IOError:
            config = {}
        super().__init__(config)

    def save(self):
        try:
            with open(self.path, 'w') as f:
                json.dump(self, f)
        except IOError as exc:
            raise CliError('Could not save GCP config: ' + str(exc))


config = Config()


def add_command(subparsers):
    parser = subparsers.add_parser('config',
        help='Display or update GCP configuration',
        description='Display or update GCP configuration used with `fw gcp` commands.')
    config_subparsers = parser.add_subparsers(
        title='available gcp config commands', metavar='')

    show_parser = config_subparsers.add_parser('show',
        help='Show properties',
        description='Show GCP configuration.')
    show_parser.set_defaults(func=config_show)
    show_parser.set_defaults(parser=show_parser)

    set_parser = config_subparsers.add_parser('set',
        help='Set properties',
        description=SET_PARSER_DESC,
        formatter_class=argparse.RawTextHelpFormatter)
    set_parser.add_argument('properties', nargs='+', metavar='KEY=VALUE',
        help='Property key and value to set')
    set_parser.set_defaults(func=config_set)
    set_parser.set_defaults(parser=set_parser)

    set_subparser_print_help(parser, config_subparsers)
    return parser


def config_show(args):
    for key in PROPERTIES:
        print('{} = {}'.format(key, config.get(key, '')))


def config_set(args):
    for prop in args.properties:
        if '=' not in prop:
            print('Invalid argument: {} (key=value format expected)'.format(prop), file=sys.stderr)
            sys.exit(1)
        key, value = prop.split('=', 1)
        if key not in PROPERTIES:
            print('Invalid property: {}'.format(key), file=sys.stderr)
            sys.exit(1)
        config[key] = value
    config.save()
    print('Successfully updated GCP config')
    config_show(args)
