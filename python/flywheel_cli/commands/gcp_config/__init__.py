from . import load_defaults, list, set
from ...util import set_subparser_print_help


def add_commands(subparsers):

    config_parser = subparsers.add_parser('config', help='The fw gcp config command group lets you set, '
                                                         'view and unset properties used by fw commands '
                                                         'which are connected to Google Cloud')

    config_subparsers = config_parser.add_subparsers(title='Available config commands', metavar='')

    load_defaults.add_command(config_subparsers)
    list.add_command(config_subparsers)
    set.add_command(config_subparsers)

    # Link help commands
    set_subparser_print_help(config_parser, config_subparsers)
