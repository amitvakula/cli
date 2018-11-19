from . import import_folder
from . import import_template
from . import import_bruker
from . import import_dicom
from . import import_bids
from . import import_heathcare
from . import gcp_config
from . import gcp_auth
from . import gcp_bq
from . import views_run
from . import views_columns
from . import views_save
from . import views_list

from . import export_bids
from . import export_bigquery

from ..config import Config
from ..util import set_subparser_print_help, print_help


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

    # import from Google Heathcare
    parsers['import hc'] = import_heathcare.add_command(import_subparsers)

    # Link help commands
    set_subparser_print_help(parser_import, import_subparsers)

    # =====
    # export
    # =====
    parser_export = subparsers.add_parser('export', help='Export data from Flywheel')
    parsers['export'] = parser_export

    export_subparsers = parser_export.add_subparsers(title='Available export commands', metavar='')

    parsers['export bids'] = export_bids.add_command(export_subparsers)

    # export Flywheel data-view to BigQuery
    parsers['export bq'] = export_bigquery.add_command(export_subparsers)

    # Link help commands
    set_subparser_print_help(parser_export, export_subparsers)

    # =====
    # Google Cloud Platform related commands
    # =====
    parser_gcp = subparsers.add_parser('gcp', help='Google Cloud platform releated commands, set default configs etc.')
    gcp_subparsers = parser_gcp.add_subparsers(title='Available gcp commands', metavar='')

    parsers['gcp'] = parser_gcp
    parsers['gcp config'] = gcp_config.add_commands(gcp_subparsers)
    parsers['gcp auth'] = gcp_auth.add_commands(gcp_subparsers)
    parsers['gcp bq'] = gcp_bq.add_command(gcp_subparsers)

    # Link help commands
    set_subparser_print_help(parser_gcp, gcp_subparsers)

    # =====
    # Data view related commands
    # =====
    parser_views = subparsers.add_parser('data-view', help='Data views')
    views_subparsers = parser_views.add_subparsers(title='Available data view commands', metavar='')

    parsers['data-view'] = parser_views
    parsers['data-view columns'] = views_columns.add_command(views_subparsers)
    parsers['data-view run'] = views_run.add_command(views_subparsers)
    parsers['data-view save'] = views_save.add_command(views_subparsers)
    parsers['data-view list'] = views_list.add_command(views_subparsers)

    # Link help commands
    set_subparser_print_help(parser_views, views_subparsers)

    # =====
    # help commands
    # =====
    parser_help = subparsers.add_parser('help')
    parser_help.add_argument('subcommands', nargs='*')
    parser_help.set_defaults(func=print_help(parser, parsers))
