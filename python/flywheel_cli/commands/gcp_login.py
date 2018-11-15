import json
import os
import sys

from flywheel.rest import ApiException

from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

CONFIG_PATH = '~/.config/flywheel/gcp_auth.json'


def add_command(subparsers):
    parser = subparsers.add_parser('auth', help='Manage Google oauth2 credentials')
    subsubparsers = parser.add_subparsers(title='Available commands', metavar='')

    login_parser = subsubparsers.add_parser('login',
                                            help='Authorize fw cli to access the Cloud Platform '
                                                 'with Google user credential')

    login_parser.set_defaults(func=login)
    login_parser.set_defaults(parser=login_parser)

    return parser


def login(args):
    fw = SdkUploadWrapper(create_flywheel_client())
    try:
        resp = fw.call_api('/gcp/auth', 'GET', response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)

    print('Go to the following link in your browser:')
    print('    {}'.format(resp['url']))

    code = input('Enter verification code: ')

    try:
        resp = fw.call_api('/gcp/auth/token', 'POST', body={'code': code}, response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)

    path = os.path.expanduser(CONFIG_PATH)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w') as f:
        json.dump(resp, f)

    print('Successfully logged in')
