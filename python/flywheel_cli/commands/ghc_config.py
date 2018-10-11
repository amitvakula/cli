from ..config import GHCConfig


def add_command(subparsers):
    parser = subparsers.add_parser('config', help='Update and show GHC config')
    parser.add_argument('--project', help='Google Cloud Platform project name')
    parser.add_argument('--token', help='Google Cloud Platform access token')
    parser.add_argument('--location', help='Google Healthcare API region name')
    parser.add_argument('--dataset', help='Google Healthcare API dataset id')
    parser.add_argument('--dicomstore', help='Google Healthcare API dicomstore id')

    parser.set_defaults(func=ghc_config)
    parser.set_defaults(parser=parser)

    return parser


def ghc_config(args):
    config = GHCConfig()
    keys = ('project', 'token', 'location', 'dataset', 'dicomstore')
    update = {key: getattr(args, key) for key in keys if getattr(args, key)}
    if update:
        print('Updating GHC config...')
        config.update(update)
        config.save()
    for key, value in config.items():
        print('{}: {}'.format(key, value))
