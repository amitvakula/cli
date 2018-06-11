
from ..importers.bruker_scan import create_bruker_scanner
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

def add_command(subparsers):
    parser = subparsers.add_parser('bruker', help='Import structured bruker data')
    parser.add_argument('folder', help='The path to the folder to import')
    parser.add_argument('group', metavar='<id>', help='The id of the group')
    parser.add_argument('project', metavar='<label>', help='The label of the project')

    parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')

    parser.set_defaults(func=import_bruker_folder)
    parser.set_defaults(parser=parser)

    return parser

def import_bruker_folder(args):
    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    # Build the importer instance
    importer = create_bruker_scanner(resolver, args.group, args.project, args.symlinks)

    # Perform the import
    importer.interactive_import(args.folder, resolver)

