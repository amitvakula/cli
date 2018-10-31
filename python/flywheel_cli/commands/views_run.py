import json
import pprint
import csv

from flywheel.api import ViewsApi

from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('run', help='Execute and adhoc or a saved data view')

    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--json', help='Data view spec')
    group.add_argument('--id', help='Saved data view id')

    group2 = parser.add_mutually_exclusive_group(required=True)
    group2.add_argument('--container-id', help='Container id to run against the data view')
    group2.add_argument('--container-path', nargs='+', help='Path to the container to run against the data view as '
                                                            'a list separated by spaces (<project-id> <project-label> '
                                                            '<subject-label> <session-label> <acquisition-label>)')

    parser.add_argument('-f', '--format',
                        help='Output format (one of: json, json-row-column, csv, tsv) (default: json)',
                        default='json')

    parser.set_defaults(func=adhoc_view)
    parser.set_defaults(parser=parser)

    return parser


def adhoc_view(args):
    fw = create_flywheel_client()
    views_api = ViewsApi(fw.api_client)

    if args.container_id:
        container_id = args.container_id
    else:
        resp = fw.resolve_path({'path': args.container_path})
        container_id = resp['path'][-1]['_id']

    if args.json:
        print('Executing adhoc view...')
        data = views_api.evaluate_view_adhoc(container_id,
                                             body=json.loads(args.json),
                                             format=args.format,
                                             _preload_content=False,
                                             _return_http_data_only=True).content.decode('utf-8')
    else:
        print('Executing saved view...')
        data = views_api.evaluate_view(args.id,
                                       container_id,
                                       format=args.format, _preload_content=False,
                                       _return_http_data_only=True).content.decode('utf-8')
    if args.format in ['json', 'json-row-column']:
        pprint.pprint(json.loads(data))
    else:
        print(data)
