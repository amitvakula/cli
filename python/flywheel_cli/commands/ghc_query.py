from ..sdk_impl import create_flywheel_client
from ..config import GHCConfig
from ..ghc import result_printer

def add_command(subparsers):
    parser = subparsers.add_parser('query', help='Query dicom files using BigQuery')
    parser.add_argument('where', nargs="+", metavar='WHERE CLAUSE', help='Where clause of the sql query')
    parser.add_argument('--output', '-o', choices=['study', 'series'], help='show only studies or series in output')

    parser.set_defaults(func=run_query)
    parser.set_defaults(parser=parser)

    return parser


def run_query(args):
    print("Running query...")
    config = GHCConfig()
    missing_fields = config.validate()
    if missing_fields:
        print('%s required, did you run `fw ghc init`?' % (', '.join(missing_fields)))
        exit(1)

    fw = create_flywheel_client()
    resp = fw.api_client.call_api('/ghc/query', 'POST', body={
                                      'location': 'US',
                                      'dataset': config.config['dataset'],
                                      'store': config.config['store'],
                                      'where': ' '.join(args.where)
                                  },
                                  auth_settings=['ApiKey'],
                                  response_type=object,
                                  _return_http_data_only=True)


    print('Query result:')
    print('Query job ID: %s' % resp['JobId'])
    print('Total number of studies: %d' % resp['TotalNumberOfStudies'])
    print('Total number of series: %d' % resp['TotalNumberOfSeries'])
    print('Total number of instances: %d' % resp['TotalNumberOfInstances'])

    printer = result_printer.ResultTreePrinter(resp)
    print(printer.generate_tree(skip_series=args.output == 'study'))

    config.set('latestJobId', resp['JobId'])
