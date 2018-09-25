from ..sdk_impl import create_flywheel_client
from ..config import GHCConfig


def add_command(subparsers):
    parser = subparsers.add_parser('import', help='Import dicom files from GHC')
    parser.add_argument('job_id', metavar='[QUERY JOB ID]', help='Job id of the query')

    parser.set_defaults(func=run_import)
    parser.set_defaults(parser=parser)

    return parser


def run_import(args):
    print("Starting import...")
    config = GHCConfig()
    missing_fields = config.validate()
    if missing_fields:
        print('%s required, did you run `fw ghc login` and `fw ghc use`?' % (', '.join(missing_fields)))
        exit(1)

    fw = create_flywheel_client()
    resp = fw.api_client.call_api('/ghc/import', 'POST', body={
                                      'project': config.config['project'],
                                      'location': config.config['location'],
                                      'dataset': config.config['dataset'],
                                      'store': config.config['store'],
                                      'jobId': args.job_id
                                  },
                                  auth_settings=['ApiKey'],
                                  response_type=object,
                                  _return_http_data_only=True)
    print('Job Id: %s' % resp['_id'])
