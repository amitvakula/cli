import json
import pprint
import csv

from flywheel.api import ViewsApi

from ..sdk_impl import create_flywheel_client


def add_command(subparsers):
    parser = subparsers.add_parser('run', help='Execute and adhoc or a saved data view')

    group = parser.add_mutually_exclusive_group()
    group.add_argument('--json', help='Data view spec')
    group.add_argument('--id', help='Saved data view id')

    parser.add_argument('-c', '--container', help='Container id to run against the data view')
    parser.add_argument('-f', '--format',
                        help='Output format (one of: json, json-row-column, csv, tsv) (default: json)',
                        default='json')

    parser.set_defaults(func=adhoc_view)
    parser.set_defaults(parser=parser)

    return parser


def adhoc_view(args):
    views_api = ViewsApi(create_flywheel_client().api_client)

    if args.json:
        print('Executing adhoc view...')
        data = views_api.evaluate_view_adhoc(args.container, body=json.loads(args.json), format=args.format, _preload_content=False, _return_http_data_only=True).content.decode('utf-8')
    else:
        print('Executing saved view...')
        data = views_api.evaluate_view(args.id, args.container,
                                       format=args.format, _preload_content=False,
                                       _return_http_data_only=True).content.decode('utf-8')
    if args.format in ['json', 'json-row-column']:
        pprint.pprint(json.loads(data))
    else:
        print(data)
