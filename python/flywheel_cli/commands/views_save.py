import json

from flywheel.api import ViewsApi
from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('save', help='Save a data view specification in Flywheel')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--json', help='Data view json spec')
    group.add_argument('--columns', nargs='+', help='Columns list separated by space to add to the data view spec')

    parser.add_argument('--label', help='Label of the data view')
    parser.add_argument('--file-container', help='File spec container')
    parser.add_argument('--analysis-label', help='File spec analysis label')
    parser.add_argument('--file-pattern', help='File spec filter pattern')

    parser.add_argument('--parent', help='Parent container id (group, project, user)', required=True)

    parser.set_defaults(func=save_view)
    parser.set_defaults(parser=parser)

    return parser


def save_view(args):
    views_api = ViewsApi(create_flywheel_client().api_client)

    if args.json:
        view_spec = json.loads(args.json)
    else:
        view_spec = {'columns': []}
        for col in args.columns:
            view_spec['columns'].append({'src': col})

    if args.file_container and args.file_pattern:
        view_spec['fileSpec'] = {}
        view_spec['fileSpec']['container'] = args.file_container
        view_spec['fileSpec']['filter'] = {
            'value': args.file_pattern
        }
        if args.analysis_label:
            view_spec['fileSpec']['analysisFilter'] = {
                'label': {'value': args.analysis_label}
            }

    if args.label:
        view_spec['label'] = args.label

    data = views_api.add_view(args.parent,
                              view_spec,
                              _preload_content=False,
                              _return_http_data_only=True).json()

    print('Successfully saved data view, id: {}'.format(data['_id']))
