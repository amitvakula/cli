import copy
import io
import re

import fs

from .import_folder import perform_folder_import

from ..importers import FolderImporter, StringMatchNode
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

def add_command(subparsers):
    parser = subparsers.add_parser('bruker', help='Import structured bruker data')
    parser.add_argument('folder', help='The path to the folder to import')
    parser.add_argument('--group', '-g', metavar='<id>', required=True, help='The id of the group')
    parser.add_argument('--project', '-p', metavar='<label>', required=True, help='The label of the project')

    parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')

    parser.set_defaults(func=import_bruker_folder)
    parser.set_defaults(parser=parser)

    return parser

def import_bruker_folder(args):
    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    # Build the importer instance
    importer = FolderImporter(resolver, group=args.group, project=args.project)

    importer.add_template_node(
        StringMatchNode(re.compile(r'(?P<session>[-\w]+)-\d+-(?P<subject>\d+)\..*'))
    )

    importer.add_composite_template_node([
        StringMatchNode(re.compile('AdjResult'), packfile_type='zip'),
        StringMatchNode('acquisition', packfile_type='pv5')
    ])

    # Perform the import
    perform_folder_import(resolver, importer, args)

