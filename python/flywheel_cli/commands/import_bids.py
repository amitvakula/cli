from ..sdk_impl import create_flywheel_client

def add_command(subparsers):
    parser = subparsers.add_parser('bids', help='Import a structured folder')
    parser.add_argument('folder', metavar='[folder]', help='The path to the folder to import')
    parser.add_argument('group', metavar='[group]', help='The id of the group')
    parser.add_argument('--project', '-p', metavar='<label>', help='Label of project to import into')
    parser.add_argument('--subject', default=None, help='Only upload data from single subject folder (i.e. sub-01)')
    parser.add_argument('--session', default=None, help='Only upload data from single session fodler (i.e. ses-01)')

    parser.set_defaults(func=import_bids)
    parser.set_defaults(parser=parser)

    return parser

def import_bids(args):
    import flywheel_bids.upload_bids

    fw = create_flywheel_client()
    flywheel_bids.upload_bids.upload_bids(fw, args.folder, args.group, project_label=args.project, validate=False,
                                          subject_label=args.subject, session_label=args.session)

