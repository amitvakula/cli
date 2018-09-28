from ..sdk_impl import create_flywheel_client
from ..config import GHCConfig
import time
from flywheel.api import JobsApi
import sys


def add_command(subparsers):
    parser = subparsers.add_parser('logs', help='Fetch the logs of a GHC import job')
    parser.add_argument('job_id', metavar='JOB_ID', help='Id of the import job')
    parser.add_argument('--follow', '-f', action='store_true', help='Follow log output')
    parser.add_argument('--tail', metavar='N', default='all',
                        help='Number of lines to show from the end of the logs (default "all")')

    parser.set_defaults(func=run_import)
    parser.set_defaults(parser=parser)

    return parser


def run_import(args):
    job_id = args.job_id
    fw = create_flywheel_client()
    jobs_api = JobsApi(fw.api_client)

    logs = jobs_api.get_job_logs(job_id)['logs']
    if args.tail == 'all':
        last_log_entry = 0
    else:
        last_log_entry = len(logs) - int(args.tail) if len(logs) > int(args.tail) else 0

    for i in range(last_log_entry, len(logs)):
        sys.stdout.write(logs[i]['msg'])
        last_log_entry = i + 1

    while args.follow:
        time.sleep(1)
        job = jobs_api.get_job(job_id)
        logs = jobs_api.get_job_logs(job_id)['logs']

        for i in range(last_log_entry, len(logs)):
            sys.stdout.write(logs[i]['msg'])
            last_log_entry = i + 1

        if job.state in ['failed', 'complete']:
            break