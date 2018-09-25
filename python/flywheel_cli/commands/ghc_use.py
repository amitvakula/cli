from ..config import GHCConfig


def add_command(subparsers):
    parser = subparsers.add_parser('use', help='Setup location, dataset, store')
    parser.add_argument('--location', '-l', help='GHC location')
    parser.add_argument('--dataset', '-d', help='GHC dataset')
    parser.add_argument('--store', '-s', help='GHC store')

    parser.set_defaults(func=run_use)
    parser.set_defaults(parser=parser)

    return parser


def run_use(args):
    config = GHCConfig()

    for key in ['location', 'dataset', 'store']:
        if getattr(args, key):
            config.set(key, getattr(args, key))
            print('Updated %s' % key)

    missing_fields = set(config.validate()).intersection(['location', 'dataset', 'store'])
    if missing_fields:
        print('%s required' % (', '.join(missing_fields)))
        exit(1)
