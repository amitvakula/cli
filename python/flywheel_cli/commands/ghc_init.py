from ..config import GHCConfig


def add_command(subparsers):
    parser = subparsers.add_parser('init', help='Setup project, access token, location, dataset and datastore for ghc commands')
    parser.add_argument('--project', '-p', help='Google project name')
    parser.add_argument('--token', '-t', help='Google auth token')
    parser.add_argument('--location', '-l', help='GHC location')
    parser.add_argument('--dataset', '-d', help='GHC dataset')
    parser.add_argument('--store', '-s', help='GHC store')

    parser.set_defaults(func=run_login)
    parser.set_defaults(parser=parser)

    return parser


def run_login(args):
    config = GHCConfig()

    for key in ['project', 'token', 'location', 'dataset', 'store']:
        if getattr(args, key):
            config.set(key, getattr(args, key))
            print('Updated %s' % key)

    missing_fields = set(config.validate()).intersection(['project', 'token', 'location', 'dataset', 'store'])
    if missing_fields:
        print('%s required' % (', '.join(missing_fields)))
        exit(1)
