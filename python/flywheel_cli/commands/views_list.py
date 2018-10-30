import json
import pprint

from flywheel.api import ViewsApi

from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('list', help='List data views of a given container')
    parser.add_argument('container_id', help='Parent container id (group, project, user)')

    parser.set_defaults(func=list_views)
    parser.set_defaults(parser=parser)

    return parser


def list_views(args):
    views_api = ViewsApi(create_flywheel_client().api_client)

    data = views_api.get_views(args.container_id, _preload_content=False, _return_http_data_only=True).content.decode('utf-8')
    pprint.pprint(json.loads(data))
