import pprint

from flywheel.api import ViewsApi

from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('columns', help='Get columns')

    parser.set_defaults(func=get_columns)
    parser.set_defaults(parser=parser)

    return parser


def get_columns(args):
    views_api = ViewsApi(create_flywheel_client().api_client)
    pprint.pprint((views_api.get_view_columns()))
