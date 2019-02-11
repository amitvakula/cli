import argparse
import sys

from .auth import get_token
from .flywheel_gcp import GCP
from .profile import get_profile
from ...errors import CliError

QUERY_FHIR_DESC = """
Search for FHIR resources in Healthcare API.
You can use FHIR's query language. See: https://www.hl7.org/fhir/search.html

examples:
  fw gcp query fhir --dataset ds --fhirstore fs --type Patient 'family=Laird'
  fw gcp query fhir --type Patient 'family=Laird'
"""


def add_command(subparsers):
    parser = subparsers.add_parser('fhir',
                                   help='Query FHIR resources in Healthcare API',
                                   description=QUERY_FHIR_DESC,
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
    parser.add_argument('--type', metavar='TYPE', help='FHIR resource type')
    parser.add_argument('query', metavar='QUERY', nargs=argparse.REMAINDER,
                        help='FHIR search query')

    parser.set_defaults(func=query_fhir)
    parser.set_defaults(parser=parser)
    return parser


def query_fhir(args):
    for param in ['project', 'location', 'dataset', 'fhirstore']:
        if not getattr(args, param, None):
            raise CliError(param + ' required')
    query = ''.join(args.query)
    gcp = GCP(get_token)
    resp = gcp.hc.list_fhir_resources(args.project, args.location, args.dataset, args.fhirstore, args.type, query)
    refs = []
    for resource in resp['entry']:
        refs.append('{}/{}'.format(resource['resource']['resourceType'], resource['resource']['id']))
    summary = 'Query matched {} resources'.format(len(refs))
    print(summary, file=sys.stderr)
    for ref in refs:
        print(ref)
