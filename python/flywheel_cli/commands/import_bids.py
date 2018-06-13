import flywheel_bids.upload_bids

from ..sdk_impl import create_flywheel_client

def add_command(subparsers):
    parser = subparsers.add_parser('bids', help='Import a structured folder')
    parser.add_argument('folder', help='The path to the folder to import')
    parser.add_argument('group', metavar='<id>', help='The id of the group')
    parser.add_argument('--project', '-p', metavar='<label>', help='Label of project to import into')

    parser.set_defaults(func=import_bids)
    parser.set_defaults(parser=parser)

    return parser

def import_bids(args):
    fw = create_flywheel_client()

    flywheel_bids.upload_bids.upload_bids(fw, args.folder, args.group, project_label=args.project, validate=False)

