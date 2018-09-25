from ..config import GHCConfig


def add_command(subparsers):
    parser = subparsers.add_parser('login', help='Setup project and access token for ghc commands')
    parser.add_argument('--project', '-p', help='Google project name')
    parser.add_argument('--token', '-t', help='Google auth token')

    parser.set_defaults(func=run_login)
    parser.set_defaults(parser=parser)

    return parser


def run_login(args):
    config = GHCConfig()

    for key in ['project', 'token']:
        if getattr(args, key):
            config.set(key, getattr(args, key))
            print('Updated %s' % key)

    missing_fields = set(config.validate()).intersection(['project', 'token'])
    if missing_fields:
        print('%s required' % (', '.join(missing_fields)))
        exit(1)
