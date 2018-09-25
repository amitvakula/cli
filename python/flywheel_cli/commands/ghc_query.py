from ..sdk_impl import create_flywheel_client
from ..config import GHCConfig


def add_command(subparsers):
    parser = subparsers.add_parser('query', help='Query dicom files using BigQuery')
    parser.add_argument('where', metavar='[WHERE CLAUSE]', help='Where clause of the sql query')

    parser.set_defaults(func=run_query)
    parser.set_defaults(parser=parser)

    return parser


def run_query(args):
    print("Running query...")
    config = GHCConfig()
    missing_fields = config.validate()
    if missing_fields:
        print('%s required, did you run `fw ghc login` and `fw ghc use`?' % (', '.join(missing_fields)))
        exit(1)

    fw = create_flywheel_client()
    resp = fw.api_client.call_api('/ghc/query', 'POST', body={
                                      'dataset': config.config['dataset'],
                                      'store': config.config['store'],
                                      'where': args.where
                                  },
                                  auth_settings=['ApiKey'],
                                  response_type=object,
                                  _return_http_data_only=True)

    row_format = []
    if len(resp['rows']) > 0 and resp['rows'][0]:
        for i, v in enumerate(resp['rows'][0]):
            row_format.append('{:^%s}' % max((len(v) + 4), (len(resp['columns'][i]) + 4)))
    row_format = '|'.join(row_format)
    header = row_format.format(*resp['columns'])
    print(header)
    print('-' * len(header))
    for row in resp['rows']:
        print(row_format.format(*row))
    print('-' * len(header))

    print('Query job ID: %s' % resp['jobId'])
