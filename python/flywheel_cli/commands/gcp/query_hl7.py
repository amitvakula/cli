import argparse
import sys

from .auth import get_token
from .flywheel_gcp import GCP
from .profile import get_profile
from ...errors import CliError

QUERY_HL7_DESC = """
Search for HL7 messages in Healthcare API.
You can use Healthcare API provided query language. See the filter query param in the docs:
https://cloud.google.com/healthcare/docs/reference/rest/v1alpha2/projects.locations.datasets.hl7V2Stores.messages/list

examples:
  fw gcp query hl7 --dataset ds --hl7store hl7 'PatientId("XYZ", "MR")'
  fw gcp query hl7 'PatientId("XYZ", "MR")'
"""


def add_command(subparsers):
    parser = subparsers.add_parser('hl7',
                                   help='Query HL7 messages in Healthcare API',
                                   description=QUERY_HL7_DESC,
                                   formatter_class=argparse.RawTextHelpFormatter)

    profile = get_profile()
    project = profile.get('project')
    location = profile.get('location')
    dataset = profile.get('hc_dataset')
    hl7store = profile.get('hc_hl7store')

    parser.add_argument('--project', metavar='NAME', default=project,
                        help='GCP project (default: {})'.format(project))
    parser.add_argument('--location', metavar='NAME', default=location,
                        help='Location (default: {})'.format(location))
    parser.add_argument('--dataset', metavar='NAME', default=dataset,
                        help='Dataset (default: {})'.format(dataset))
    parser.add_argument('--hl7store', metavar='NAME', default=hl7store,
                        help='HL7 store (default: {})'.format(hl7store))
    parser.add_argument('query', metavar='QUERY', nargs=argparse.REMAINDER,
                        help='HL7 filter query')

    parser.set_defaults(func=query_hl7)
    parser.set_defaults(parser=parser)
    return parser


def query_hl7(args):
    for param in ['project', 'location', 'dataset', 'hl7store']:
        if not getattr(args, param, None):
            raise CliError(param + ' required')
    query = ''.join(args.query)
    gcp = GCP(get_token)
    resp = gcp.hc.list_hl7_messages(args.project, args.location, args.dataset, args.hl7store, query)
    ids = list(map(lambda x: x.split('/')[-1], resp['messages']))
    summary = 'Query matched {} resources'.format(len(ids))
    print(summary, file=sys.stderr)
    for ref in ids:
        print(ref)
