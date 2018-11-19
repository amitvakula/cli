import sys

from flywheel.rest import ApiException

from ...sdk_impl import create_flywheel_client, SdkUploadWrapper


def add_command(subparsers):
    parser = subparsers.add_parser('login', help='Authorize fw cli to access the Cloud Platform '
                                                 'with Google user credential')

    parser.set_defaults(func=login)
    parser.set_defaults(parser=parser)

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

    print('Successfully logged in')
