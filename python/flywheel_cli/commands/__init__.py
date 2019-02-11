from . import import_folder
from . import import_template
from . import import_bruker
from . import import_dicom
from . import import_bids
from . import import_parrec

from . import export_bids

from . import retry_job

from . import view
from . import gcp

from ..config import Config
from ..util import set_subparser_print_help, print_help


def get_config(args):
    args.config = Config(args)

def add_commands(parser):
    # Setup global configuration args
    parser.set_defaults(config=get_config)

    global_parser = Config.get_global_parser()
    import_parser = Config.get_import_parser()
    deid_parser = Config.get_deid_parser()

    # map commands for help function
    parsers = {}

    # Create subparsers
    subparsers = parser.add_subparsers(title='Available commands', metavar='')

    # =====
    # import
    # =====
    parser_import = subparsers.add_parser('import', help='Import data into Flywheel')

    parsers['import'] = parser_import

    import_subparsers = parser_import.add_subparsers(title='Available import commands', metavar='')

    # import folder
    parsers['import folder'] = import_folder.add_command(import_subparsers, [global_parser, import_parser, deid_parser])

    # import bids
    parsers['import bids'] = import_bids.add_command(import_subparsers, [global_parser])

    # import dicom 
    parsers['import dicom'] = import_dicom.add_command(import_subparsers, [global_parser, import_parser, deid_parser])

    # import bruker
    parsers['import bruker'] = import_bruker.add_command(import_subparsers, [global_parser, import_parser])

    # import parrec
    parsers['import parrec'] = import_parrec.add_command(import_subparsers, [global_parser, import_parser])

    # import template
    parsers['import template'] = import_template.add_command(import_subparsers, [global_parser, import_parser, deid_parser])

    # Link help commands
    set_subparser_print_help(parser_import, import_subparsers)


    # =====
    # export
    # =====
    parser_export = subparsers.add_parser('export', help='Export data from Flywheel')
    parsers['export'] = parser_export

    export_subparsers = parser_export.add_subparsers(title='Available export commands', metavar='')

    parsers['export bids'] = export_bids.add_command(export_subparsers, [global_parser])

    # Link help commands
    set_subparser_print_help(parser_export, export_subparsers)


    # =====
    # job
    # =====
    parser_job = subparsers.add_parser('job', help='Start or manage server jobs')
    parser_job.set_defaults(config=get_config)
    parsers['job'] = parser_job

    job_subparsers = parser_job.add_subparsers(title='Available job commands', metavar='')

    parsers['job retry'] = retry_job.add_command(job_subparsers, [global_parser])

    # Link help commands
    set_subparser_print_help(parser_job, job_subparsers)


    # =====
    # view
    # =====
    parser_view = subparsers.add_parser('view', help='Execute and save Flywheel data-views')
    parsers['view'] = parser_view
    view_subparsers = parser_view.add_subparsers(title='Available data-view commands', metavar='')

    parsers['view run'] = view.add_run_command(view_subparsers)
    parsers['view ls'] = view.add_ls_command(view_subparsers)
    parsers['view cols'] = view.add_cols_command(view_subparsers)

    # Link help commands
    set_subparser_print_help(parser_view, view_subparsers)

    # =====
    # gcp - Google Cloud Platform integration commands
    # =====
    parser_gcp_help = 'Google Cloud Platform integration commands'
    parser_gcp = subparsers.add_parser('gcp', help=parser_gcp_help, description=parser_gcp_help)
    parsers['gcp'] = parser_gcp
    gcp_subparsers = parser_gcp.add_subparsers(title='Available gcp commands', metavar='')

    parsers['gcp auth'] = gcp.auth.add_command(gcp_subparsers)
    parsers['gcp profile'] = gcp.profile.add_command(gcp_subparsers)

    parser_gcp_query_help = 'Query and display Healthcare API data available on GCP'
    parser_gcp_query = gcp_subparsers.add_parser('query', help=parser_gcp_query_help, description=parser_gcp_query_help)
    parsers['gcp query'] = parser_gcp_query
    gcp_query_subparsers = parser_gcp_query.add_subparsers(title='available gcp query commands', metavar='')
    parsers['gcp query dicom'] = gcp.query_dicom.add_command(gcp_query_subparsers)
    parsers['gcp query fhir'] = gcp.query_fhir.add_command(gcp_query_subparsers)
    parsers['gcp query hl7'] = gcp.query_hl7.add_command(gcp_query_subparsers)
    set_subparser_print_help(parser_gcp_query, gcp_query_subparsers)

    parser_gcp_import_help = 'Import data from GCP into Flywheel'
    parser_gcp_import = gcp_subparsers.add_parser('import', help=parser_gcp_import_help, description=parser_gcp_import_help)
    parsers['gcp import'] = parser_gcp_import
    gcp_import_subparsers = parser_gcp_import.add_subparsers(title='available gcp import commands', metavar='')
    parsers['gcp import dicom'] = gcp.import_dicom.add_command(gcp_import_subparsers)
    parsers['gcp import fhir'] = gcp.import_fhir.add_command(gcp_import_subparsers)
    parsers['gcp import hl7'] = gcp.import_hl7.add_command(gcp_import_subparsers)
    set_subparser_print_help(parser_gcp_import, gcp_import_subparsers)

    parser_gcp_export = gcp_subparsers.add_parser('export', help='export data from Flywheel into GCP')
    parsers['gcp export'] = parser_gcp_export
    gcp_export_subparsers = parser_gcp_export.add_subparsers(title='Available GCP export commands', metavar='')
    parsers['gcp export view'] = gcp.export_view.add_command(gcp_export_subparsers)
    set_subparser_print_help(parser_gcp_export, gcp_export_subparsers)

    # Link help commands
    set_subparser_print_help(parser_gcp, gcp_subparsers)

    # =====
    # help commands
    # =====
    parser_help = subparsers.add_parser('help')
    parser_help.add_argument('subcommands', nargs='*')
    parser_help.set_defaults(func=print_help(parser, parsers))

    # Finally, set default values for all parsers
    Config.set_defaults(parsers)
