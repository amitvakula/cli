import math

from . import import_folder
from . import import_template
from . import import_bruker
from . import import_dicom
from . import import_bids

from . import export_bids

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

def config_import(args):
    # Set the default compression (used by zipfile/ZipFS)
    import zlib
    zlib.Z_DEFAULT_COMPRESSION = args.compression_level

    if args.jobs == -1:
        import multiprocessing
        args.jobs = max(1, math.floor(multiprocessing.cpu_count() / 2))

def add_commands(parser, legacy_commands):
    # map commands for help function
    parsers = {}

    # Create subparsers
    subparsers = parser.add_subparsers(title='Available commands', metavar='')

    # =====
    # import
    # =====
    parser_import = subparsers.add_parser('import', help='Import data into Flywheel')
    compression_levels = [-1] + list(range(9))
    parser_import.add_argument('--jobs', '-j', default=-1, type=int, help='The number of concurrent jobs to run (e.g. compression jobs)')
    parser_import.add_argument('--compression-level', default=1, type=int, choices=compression_levels, 
            help='The compression level to use for packfiles')
    parser_import.set_defaults(config=config_import)

    parsers['import'] = parser_import

    import_subparsers = parser_import.add_subparsers(title='Available import commands', metavar='')
    set_subparser_print_help(parser_import)

    # import folder
    parsers['import folder'] = import_folder.add_command(import_subparsers)

    # import bids
    parsers['import bids'] = import_bids.add_command(import_subparsers)

    # import dicom 
    parsers['import dicom'] = import_dicom.add_command(import_subparsers)

    # import bruker
    parsers['import bruker'] = import_bruker.add_command(import_subparsers)

    # import template
    parsers['import template'] = import_template.add_command(import_subparsers)

    # =====
    # export
    # =====
    parser_export = subparsers.add_parser('export', help='Export data from Flywheel')
    parsers['export'] = parser_export

    export_subparsers = parser_export.add_subparsers(title='Available export commands', metavar='')
    set_subparser_print_help(parser_export)

    parsers['export bids'] = export_bids.add_command(export_subparsers)

    # =====
    # cli subcommands (stubs)
    # =====
    for cmd, desc in legacy_commands.items():
        subparsers.add_parser(cmd, help=desc)

    # =====
    # help commands
    # =====
    parser_help = subparsers.add_parser('help')
    parser_help.add_argument('subcommands', nargs='*')
    parser_help.set_defaults(func=print_help(parser, parsers))



