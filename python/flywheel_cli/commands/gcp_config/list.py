from ...config import GCPConfig


def add_command(subparsers):
    parser = subparsers.add_parser('list', help='List properties')
    parser.set_defaults(func=list_config)
    parser.set_defaults(parser=parser)

    return parser


def list_config(args):
    config = GCPConfig()

    for k, v in config.items():
        print('[{}]'.format(k))
        for kk, vv in v.items():
            print('{} = {}'.format(kk, vv))
        print('')
