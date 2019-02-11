import argparse
import sys
import time

from .auth import get_token_id
from .profile import get_profile
from ...errors import CliError
from ...sdk_impl import create_flywheel_client, create_flywheel_session

IMPORT_FHRI_DESC = """
Import FHIR resources from Google Healthcare API by resource ref. Resource reference format is the following:
<type>/<id>.
Read resource refs from STDIN if none were given as arguments. To discover FHIRs
available in GCP for import see `fw gcp query fhir help`.

examples:
  fw gcp import fhir --ref Patient/60433200-2aec-4eab-a226-31add8752b32 group/project
  fw gcp query fhir --refs 'family=Laird' --type 'Patient' | fw gcp import fhir group/project

"""


def add_command(subparsers):
    parser = subparsers.add_parser('fhir',
                                   help='Import FHIR resources from Google Healthcapiare API by references',
                                   description=IMPORT_FHRI_DESC,
                                   formatter_class=argparse.RawTextHelpFormatter)

    profile = get_profile()
    project = profile.get('project')
    location = profile.get('location')
    dataset = profile.get('hc_dataset')
    fhirstore = profile.get('hc_fhirstore')

    parser.add_argument('--project', metavar='NAME', default=project,
                        help='GCP project (default: {})'.format(project))
    parser.add_argument('--location', metavar='NAME', default=location,
                        help='Location (default: {})'.format(location))
    parser.add_argument('--dataset', metavar='NAME', default=dataset,
                        help='Dataset (default: {})'.format(dataset))
    parser.add_argument('--fhirstore', metavar='NAME', default=fhirstore,
                        help='FHIR store (default: {})'.format(fhirstore))
    parser.add_argument('--ref', metavar='REF', action='append', dest='refs', default=[],
                        help='<type>/<resource_id> to import')
    parser.add_argument('--async', action='store_true',
                        help='Do not wait for import job to finish')
    parser.add_argument('--debug', action='store_true',
                        help='Run import gear in debug mode')
    parser.add_argument('container', metavar='CONTAINER',
                        help='Flywheel project to import into (group/proj)')

    parser.set_defaults(func=import_fhir)
    parser.set_defaults(parser=parser)
    return parser


def import_fhir(args):
    # TODO fw de-identify profiles
    # TODO gcp de-identify
    for param in ['project', 'location', 'dataset', 'fhirstore']:
        if not getattr(args, param, None):
            raise CliError(param + ' required')

    refs = args.refs or sys.stdin.read().split()
    if not refs:
        raise CliError('At least one ref required.')
    api = create_flywheel_session()
    for gear in api.get('/gears'):
        if gear['gear']['name'] == 'ghc-import':
            break
    else:
        raise CliError('ghc-import gear is not installed on ' + api.baseurl.replace('/api', ''))
    client = create_flywheel_client()
    # TODO check whether it is a project
    project = client.lookup(args.container)
    job = api.post('/jobs/add', json={
        'gear_id': gear['_id'],
        'destination': {'type': 'project', 'id': project.id},
        'config': {
            'auth_token_id': get_token_id(),
            'gcp_project': args.project,
            'hc_location': args.location,
            'hc_dataset': args.dataset,
            'hc_fhirstore': args.fhirstore,
            'fhir_resource_refs': refs,
            'project_id': project.id,
            'log_level': 'DEBUG' if args.debug else 'INFO',
        }
    })
    job_id = job['_id']
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
