from ..sdk_impl import create_flywheel_client
from ..config import GHCConfig
import time
from flywheel.api import JobsApi
import sys

def add_command(subparsers):
    parser = subparsers.add_parser('import', help='Import dicom series from GHC')
    parser.add_argument('level', metavar='LEVEL', choices=['studies', 'series'], help='Studies or series level import')

    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--from-query', nargs='?', const='latest',
                        help='Import studies/series of a given query or the last one if query id is not specified')
    group.add_argument('--uids', nargs='+', metavar='UID', help='Import the given studies/series')

    parser.add_argument('--exclude', nargs='+', metavar='UID', help='Exclude the given studies/series')
    parser.add_argument('--logs', action='store_true',
                        help='Show the import job logs')
    parser.add_argument('--log-level', choices=['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL'], default='INFO',
                        help='Log level')
    parser.add_argument('--timeout', type=int, metavar='N', default=120,
                        help='Wait N seconds for the job to finish, default is 120 seconds')
    parser.add_argument('--async', action='store_true',
                        help='Run the import job async, in this case you will get a job id '
                             'that can be used to check the status of the job')

    parser.set_defaults(func=run_import)
    parser.set_defaults(parser=parser)

    return parser


def run_import(args):
    print("Starting import...")

    config = GHCConfig()
    missing_fields = config.validate()
    if missing_fields:
        print('%s required, did you run `fw ghc init`?' % (', '.join(missing_fields)))
        exit(1)

    payload = {
        'project': config.config['project'],
        'location': config.config['location'],
        'dataset': config.config['dataset'],
        'store': config.config['store'],
        'exclude': args.exclude,
        'log-level': args.log_level,
        'level': args.level
    }

    if args.from_query and args.from_query != 'latest':
        payload['jobId'] = args.from_query
    elif args.uids:
        payload['uids'] = args.uids
    else:
        latest_job_id = config.config.get('latestJobId')
        if not latest_job_id:
            print('We did\'t find recent query job. Did you run `fw ghc query`?')
            exit(1)
        else:
            payload['jobId'] = latest_job_id

    fw = create_flywheel_client()
    resp = fw.api_client.call_api('/ghc/import', 'POST', body=payload,
                                  auth_settings=['ApiKey'],
                                  response_type=object,
                                  _return_http_data_only=True)

    job_id = resp['_id']
    print('Job Id: %s' % job_id)
    if args.async:
        exit(0)

    print('Waiting for job to finish...')
    jobs_api = JobsApi(fw.api_client)
    start = time.time()
    last_log_entry = 0
    while True:
        job = jobs_api.get_job(job_id)
        time.sleep(1)
        if args.logs:
            logs = jobs_api.get_job_logs(job_id)['logs']
            for i in range(last_log_entry, len(logs)):
                sys.stdout.write(logs[i]['msg'])
                last_log_entry = i + 1
        if job.state in ['failed', 'complete'] or time.time() - start > args.timeout:
            break

    print('Job %s' % job.state)
