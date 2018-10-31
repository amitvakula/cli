import json

from flywheel.api import ViewsApi
from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('save', help='Save a data view specification in Flywheel')
    parser.add_argument('json', help='Data view spec')
    parser.add_argument('--parent', help='Parent container id (group, project, user)')

    parser.set_defaults(func=save_view)
    parser.set_defaults(parser=parser)

    return parser


def save_view(args):
    views_api = ViewsApi(create_flywheel_client().api_client)

    data = views_api.add_view(args.parent,
                              json.loads(args.json),
                              _preload_content=False,
                              _return_http_data_only=True).json()

    print('Successfully saved data view, id: {}'.format(data['_id']))
