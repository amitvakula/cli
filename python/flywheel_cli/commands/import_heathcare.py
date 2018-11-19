import sys
import time

from flywheel.api import JobsApi
from flywheel.rest import ApiException
from ..config import GCPConfig
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper


def add_command(subparsers):
    parser = subparsers.add_parser('hc', help='Import dicom studies or series from Google Healthcare API')
    parser.add_argument('group', metavar='group_id', help='The id of the group')
    parser.add_argument('project', metavar='project_label', help='The label of the project')
    parser.add_argument('uids', metavar='uids', nargs='+', help='Study or series instance UIDS, '
                                                                'depends on the --study flag')

    parser.add_argument('--study', action='store_true', help='Import whole studies instead of individual series')
    parser.add_argument('--de-identify', action='store_true', help='De-identify dicoms before import')
    parser.add_argument('--async', action='store_true', help='Do not wait for import job to finish')

    parser.set_defaults(func=ghc_import)
    parser.set_defaults(parser=parser)

    return parser


def ghc_import(args):
    ghc_config = GCPConfig()
    core_keys = ('project',)
    hc_keys = ('location', 'dataset', 'dicomstore')

    payload = {key: ghc_config['core'][key] for key in core_keys if ghc_config.get('core', {}).get(key)}
    for key in hc_keys:
        if ghc_config.get('healthcare', {}).get(key):
            payload[key] = ghc_config['healthcare'][key]

    if args.study:
        payload['study'] = True

    if args.de_identify:
        payload['de_identify'] = True

    payload['uids'] = args.uids
    payload['group_id'] = args.group
    payload['project_label'] = args.project

    print('Starting import...')
    fw = SdkUploadWrapper(create_flywheel_client())
    try:
        resp = fw.call_api('/gcp/hc/import', 'POST', body=payload, response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)
    job_id = resp['_id']
    print('Job: ' + job_id)

    if not args.async:
        jobs_api = JobsApi(fw.fw.api_client)
        last_log_entry = 0
        print('Waiting for import job to finish...')
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
