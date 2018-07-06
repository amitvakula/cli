from . import import_folder
from . import import_template
from . import import_bruker
from . import import_dicom
from . import import_bids

from . import fgm_install
from . import fgm_upgrade
from . import fgm_show
from . import fgm_list
from . import fgm_search


from . import export_bids

from ..config import Config

def set_subparser_print_help(parser):
    def print_help(args):
        parser.print_help()
    parser.set_defaults(func=print_help)

def print_help(default_parser, parsers):
    def print_help_fn(args):
        subcommands = ' '.join(args.subcommands)
        if subcommands in parsers:
            parsers[subcommands].print_help()
        else:
            default_parser.print_help()

    return print_help_fn

def get_config(args):
    args.config = Config(args)

def add_commands(parser):
    # map commands for help function
    parsers = {}

    # Create subparsers
    subparsers = parser.add_subparsers(title='Available commands', metavar='')

    # =====
    # import
    # =====
    parser_import = subparsers.add_parser('import', help='Import data into Flywheel')
    parser_import.set_defaults(config=get_config)

    parsers['import'] = parser_import

    import_subparsers = parser_import.add_subparsers(title='Available import commands', metavar='')
    set_subparser_print_help(parser_import)

    # import folder
    parsers['import folder'] = import_folder.add_command(import_subparsers)
    Config.add_config_args(parsers['import folder'])

    # import bids
    parsers['import bids'] = import_bids.add_command(import_subparsers)

    # import dicom 
    parsers['import dicom'] = import_dicom.add_command(import_subparsers)
    Config.add_config_args(parsers['import dicom'])

    # import bruker
    parsers['import bruker'] = import_bruker.add_command(import_subparsers)
    Config.add_config_args(parsers['import bruker'])

    # import template
    parsers['import template'] = import_template.add_command(import_subparsers)
    Config.add_config_args(parsers['import template'])

    # =====
    # export
    # =====
    parser_export = subparsers.add_parser('export', help='Export data from Flywheel')
    parsers['export'] = parser_export

    export_subparsers = parser_export.add_subparsers(title='Available export commands', metavar='')
    set_subparser_print_help(parser_export)

    parsers['export bids'] = export_bids.add_command(export_subparsers)

    # =====
    # fgm (Flywheel Gear Manager)
    # =====
    parser_fgm = subparsers.add_parser('fgm', help='Manage gears installed on flywheel instances')
    parsers['fgm'] = parser_fgm

    fgm_subparsers = parser_fgm.add_subparsers(title='Available fgm commands', metavar='')
    set_subparser_print_help(parser_fgm)

    parsers['fgm install'] = fgm_install.add_command(fgm_subparsers)
    parsers['fgm upgrade'] = fgm_upgrade.add_command(fgm_subparsers)
    parsers['fgm show'] = fgm_show.add_command(fgm_subparsers)
    parsers['fgm list'] = fgm_list.add_command(fgm_subparsers)
    parsers['fgm search'] = fgm_search.add_command(fgm_subparsers)

    # =====
    # help commands
    # =====
    parser_help = subparsers.add_parser('help')
    parser_help.add_argument('subcommands', nargs='*')
    parser_help.set_defaults(func=print_help(parser, parsers))



