import sys
import time
import json
from flywheel.api import JobsApi
from flywheel.rest import ApiException

from ..sdk_impl import create_flywheel_client, SdkUploadWrapper


def add_command(subparsers):
    parser = subparsers.add_parser('export', help='Export flywheel data view to BigQuery')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--json', help='Data view spec')
    group.add_argument('--id', help='Saved data view id')
    group.add_argument('--columns', nargs='+', help='Columns list separated by space to add to the data view spec')

    group2 = parser.add_mutually_exclusive_group(required=True)
    group2.add_argument('--container-id', help='Container id to run against the data view')
    group2.add_argument('--container-path', nargs='+', help='Path to the container to run against the data view as '
                                                            'a list separated by spaces (<project-id> <project-label> '
                                                            '<subject-label> <session-label> <acquisition-label>)')

    parser.add_argument('-t', '--token', help='Google auth token')
    parser.add_argument('-p', '--project', help='Google project id')
    parser.add_argument('--async', action='store_true', help='Do not wait for export job to finish')

    parser.set_defaults(func=export_view)
    parser.set_defaults(parser=parser)

    return parser


def export_view(args):
    print('Starting export...')

    fw = SdkUploadWrapper(create_flywheel_client())

    if args.container_id:
        container_id = args.container_id
    else:
        resp = fw.fw.resolve_path({'path': args.container_path})
        container_id = resp['path'][-1].id

    view_spec_str = None

    if args.json:
        view_spec_str = args.json
    elif args.columns:
        view_spec = {'columns': []}
        for col in args.columns:
            view_spec['columns'].append({'src': col})

        view_spec_str = json.dumps(view_spec)

    payload = {
        'json': view_spec_str,
        'container_id': container_id,
        'token': args.token,
        'view_id': args.id
    }

    if args.project:
        payload['project'] = args.project

    try:
        resp = fw.call_api('/gcp/bq/export', 'POST', body=payload, response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)

    job_id = resp['_id']
    print('Job: ' + job_id)

    if not args.async:
        jobs_api = JobsApi(fw.fw.api_client)
        last_log_entry = 0
        print('Waiting for export job to finish...')
        while True:
            time.sleep(1)
            logs = jobs_api.get_job_logs(job_id)['logs']
            for i in range(last_log_entry, len(logs)):
                sys.stdout.write(logs[i]['msg'])
                last_log_entry = i + 1
            job = jobs_api.get_job(job_id)
            if job.state in ['failed', 'complete']:
                print('Job ' + job.state)
                break
