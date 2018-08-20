import argparse
import re
import textwrap

from ..config import Config
from ..importers import parse_template_string, FolderImporter
from ..util import set_nested_attr, split_key_value_argument, METADATA_ALIASES
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

def add_command(subparsers):
    parser = subparsers.add_parser('template', help='Import a folder, using a template',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description=textwrap.dedent("""\
            A template string can be used to extract metadata from custom folder trees.

            Simple properties can be extracted using replacement syntax; subfolders
            are delimited by ':'. For example:
                'ses-{session}-{subject}:{acquisition}'
            would extract the session label "01", subject code "ex2002" and
            acquisition label of "Diffusion" from the folder names:
                ses-01-ex2002/Diffusion

            Any variable can be set using --set-var. e.g. to set the acquisition label:
                --set-var 'acquisition.label=Scan'
        """))

    parser.add_argument('--group', '-g', metavar='<id>', help='The id of the group, if not in folder structure')
    parser.add_argument('--project', '-p', metavar='<label>', help='The label of the project, if not in folder structure')

    Config.add_deid_args(parser)

    parser.add_argument('--repack', action='store_true', help='Whether or not to validate, de-identify and repackage zipped packfiles')

    no_level_group = parser.add_mutually_exclusive_group()
    no_level_group.add_argument('--no-subjects', action='store_true', help='no subject level (create a subject for every session)')
    no_level_group.add_argument('--no-sessions', action='store_true', help='no session level (create a session for every subject)')

    parser.add_argument('--set-var', '-s', metavar='key=value', action='append', default=[], 
        type=split_key_value_argument, help='Set arbitrary key-value pairs')
    
    parser.add_argument('template', help='The template string')
    parser.add_argument('folder', help='The path to the folder to import')

    parser.set_defaults(func=import_folder_with_template)
    parser.set_defaults(parser=parser)

    return parser

def build_context(variables):
    context = {}
    for key, value in variables:
        if key in METADATA_ALIASES:
            key = METADATA_ALIASES[key]
        set_nested_attr(context, key, value)
    return context

def import_folder_with_template(args):
    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    # Build the importer instance
    importer = FolderImporter(resolver, group=args.group, project=args.project, 
        repackage_archives=args.repack, merge_subject_and_session=(args.no_subjects or args.no_sessions), 
        context=build_context(args.set_var), config=args.config)

    # Build the template string
    try:
        importer.root_node = parse_template_string(args.template) 
    except (ValueError, re.error) as e:
        args.parser.error('Invalid template: {}'.format(e))

    # Perform the import
    importer.interactive_import(args.folder, resolver)

