import json

from flywheel.api import ViewsApi
from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('save', help='Save a data view specification in Flywheel')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--json', help='Data view json spec')
    group.add_argument('--columns', nargs='+', help='Columns list separated by space to add to the data view spec')

    parser.add_argument('--parent', help='Parent container id (group, project, user)')

    parser.set_defaults(func=save_view)
    parser.set_defaults(parser=parser)

    return parser


def save_view(args):
    views_api = ViewsApi(create_flywheel_client().api_client)

    view_spec = None

    if args.json:
        view_spec = json.loads(args.json)
    elif args.columns:
        view_spec = {'columns': []}
        for col in args.columns:
            view_spec['columns'].append({'src': col})

    data = views_api.add_view(args.parent,
                              view_spec,
                              _preload_content=False,
                              _return_http_data_only=True).json()

    print('Successfully saved data view, id: {}'.format(data['_id']))
