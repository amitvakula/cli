import argparse
import collections
import json
import os
import sys

from ...errors import CliError
from ...sdk_impl import create_flywheel_session
from ...util import set_subparser_print_help


PROPERTIES = collections.OrderedDict({
    'project': 'Google Cloud Platform project',
    'location': 'API location/region (Healthcare, AutoML, etc.)',
    'hc_dataset': 'Healthcare API dataset',
    'hc_dicomstore': 'Healthcare API dicom-store',
    'hc_fhirstore': 'Healthcare API FHIR-store',
    'hc_hl7store': 'Healthcare API HL7-store',
})

PARSER_DESC = """
Create, update and delete GCP user profiles used throughout `fw gcp` commands.

examples:
  fw gcp profile list
  fw gcp profile create my-profile project=my-project location=us-central1 ...
  fw gcp profile update my-profile hc_dataset=my-dataset ...
  fw gcp profile delete my-profile

available profile properties:
""" + '\n'.join('  {:16.16} {}'.format(key, desc) for key, desc in PROPERTIES.items())


def add_command(subparsers):
    parser = subparsers.add_parser('profile',
        help='Manage GCP profiles',
        description=PARSER_DESC,
        formatter_class=argparse.RawTextHelpFormatter)
    profile_subparsers = parser.add_subparsers(
        title='available gcp profile commands', metavar='')

    list_parser = profile_subparsers.add_parser('list',
        help='List GCP profiles',
        description='List available GCP user profiles.')
    list_parser.set_defaults(func=profile_list)
    list_parser.set_defaults(parser=list_parser)

    create_parser = profile_subparsers.add_parser('create',
        help='Create a new GCP profile',
        description='Create a new GCP user profile.')
    create_parser.add_argument('name', metavar='NAME',
        help='Profile name to create')
    create_parser.add_argument('properties', nargs='+', metavar='KEY=VALUE',
        help='Property key and value')
    create_parser.set_defaults(func=profile_create)
    create_parser.set_defaults(parser=create_parser)

    update_parser = profile_subparsers.add_parser('update',
        help='Update a GCP profile',
        description='Update an existing GCP user profile.')
    update_parser.add_argument('name', metavar='NAME',
        help='Profile name to update')
    update_parser.add_argument('properties', nargs='+', metavar='KEY=VALUE',
        help='Property key and value')
    update_parser.set_defaults(func=profile_update)
    update_parser.set_defaults(parser=update_parser)

    delete_parser = profile_subparsers.add_parser('delete',
        help='Delete a GCP profile',
        description='Delete an existing GCP user profile.')
    delete_parser.add_argument('name', metavar='NAME',
        help='Profile name to delete')
    delete_parser.set_defaults(func=profile_delete)
    delete_parser.set_defaults(parser=delete_parser)

    set_subparser_print_help(parser, profile_subparsers)
    return parser


def get_profiles():
    api = create_flywheel_session()
    try:
        return api.get('/users/self/info').get('gcp_profiles', [])
    except CliError:
        return []


def get_profile():
    profiles = get_profiles()
    if profiles:
        return profiles[0]
    else:
        return {}


def args_to_profile(args):
    profile = {'name': args.name}
    for prop in args.properties:
        if '=' not in prop:
            print('Invalid argument: {} (key=value format expected)'.format(prop), file=sys.stderr)
            sys.exit(1)
        key, value = prop.split('=', 1)
        if key not in PROPERTIES:
            print('Invalid property: {}'.format(key), file=sys.stderr)
            sys.exit(1)
        profile[key] = value
    return profile


def profile_list(args):
    profiles = get_profiles()
    if profiles:
        for index, profile in enumerate(profiles):
            print('' + profile['name'] + (' (active)' if index == 0 else '') + ':')
            for key in PROPERTIES:
                print('  {} = {}'.format(key, profile.get(key, '')))
    else:
        print('No GCP user profiles found.')


def profile_create(args):
    profiles = get_profiles()
    if any(p['name'] == args.name for p in profiles):
        raise CliError('GCP profile already exists: ' + args.name)
    profiles = [args_to_profile(args)] + profiles
    api = create_flywheel_session()
    api.post('/users/self/info', json={'set': {'gcp_profiles': profiles}})
    print('Successfully created GCP profile ' + args.name)


def profile_update(args):
    profiles = get_profiles()
    for profile in profiles:
        if profile['name'] == args.name:
            break
    else:
        raise CliError('GCP profile not found: ' + args.name)
    profile.update(args_to_profile(args))
    profiles = [profile] + [p for p in profiles if p['name'] != args.name]
    api = create_flywheel_session()
    api.post('/users/self/info', json={'set': {'gcp_profiles': profiles}})
    print('Successfully created GCP profile ' + args.name)


def profile_delete(args):
    raise CliError('Not implemented')
