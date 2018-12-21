import argparse
import sys
import time

from .auth import get_token_id
from .config import config
from ...errors import CliError
from ...sdk_impl import create_flywheel_client, create_flywheel_session


IMPORT_DICOM_DESC = """
Import dicom series or whole studies from Google Healthcare API by InstanceUIDs.
Read UIDs from STDIN if none were given as arguments. To discover dicoms
available in GCP for import see `fw gcp query dicom help`.

examples:
  fw gcp import dicom --uid 1.2.840.113619.2.408... group/project
  fw gcp query dicom --uids 'AccessionNumber="AccNo"' | fw gcp import dicom group/project

"""


def add_command(subparsers):
    parser = subparsers.add_parser('dicom',
        help='Import dicoms from Google Healthcapiare API by InstanceUID',
        description=IMPORT_DICOM_DESC,
        formatter_class=argparse.RawTextHelpFormatter)

    project = config.get('project')
    location = config.get('hcapi.location')
    dataset = config.get('hcapi.dataset')
    dicomstore = config.get('hcapi.dicomstore')

    parser.add_argument('--project', metavar='NAME', default=project,
        help='GCP project (default: {})'.format(project))
    parser.add_argument('--location', metavar='NAME', default=location,
        help='Location (default: {})'.format(location))
    parser.add_argument('--dataset', metavar='NAME', default=dataset,
        help='Dataset (default: {})'.format(dataset))
    parser.add_argument('--dicomstore', metavar='NAME', default=dicomstore,
        help='Dicomstore (default: {})'.format(dicomstore))
    parser.add_argument('--uid', metavar='UID', dest='uids', default=[],
        help='Study/SeriesInstanceUID to import')
    parser.add_argument('--de-identify', action='store_true',
        help='De-identify dicoms before import')
    parser.add_argument('--async', action='store_true',
        help='Do not wait for import job to finish')
    parser.add_argument('container', metavar='CONTAINER',
        help='Flywheel project to import into (group/proj)')

    parser.set_defaults(func=import_dicom)
    parser.set_defaults(parser=parser)
    return parser


def import_dicom(args):
    # TODO fw de-identify profiles
    # TODO gcp de-identify
    uids = args.uids or sys.stdin.read().split()
    if not uids:
        raise CliError('At least one UID required.')
    api = create_flywheel_session()
    for gear in api.get('/gears'):
        if gear['gear']['name'] == 'ghc-import':
            break
    else:
        raise CliError('ghc-import gear is not installed on ' + api.baseurl.replace('/api', ''))
    client = create_flywheel_client()
    # TODO check whether it is a project
    project = client.lookup(args.container)
    resp = api.post('/jobs', json={
        'gear_id': gear['_id'],
        'destination': {'type': 'project', 'id': project.id},
        'config': {
            'auth_token_id': get_token_id(),
            'gcp_project': args.project,
            'hc_location': args.location,
            'hc_dataset': args.dataset,
            'hc_datastore': args.dicomstore,
            'uids': args.uids,
            'de_identify': args.de_identify,
            'project_id': project.id,
        }
    })
    job_id = resp.json()['_id']
    print('Started ghc-import job ' + job_id)

    if not args.async:
        jobs_api = client.jobs_api
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
