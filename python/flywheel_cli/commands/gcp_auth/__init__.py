from . import login, revoke
from ...util import set_subparser_print_help


def add_commands(subparsers):

    config_parser = subparsers.add_parser('auth', help='The fw gcp auth command group lets you grant '
                                                       'and revoke authorization to Flywheel to access '
                                                       'Google Cloud Platform.')

    config_subparsers = config_parser.add_subparsers(title='Available auth commands', metavar='')

    login.add_command(config_subparsers)
    revoke.add_command(config_subparsers)

    # Link help commands
    set_subparser_print_help(config_parser, config_subparsers)
