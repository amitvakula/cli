import sys

from flywheel.rest import ApiException

from ...sdk_impl import create_flywheel_client, SdkUploadWrapper


def add_command(subparsers):
    parser = subparsers.add_parser('revoke', help='Revoke access token. Note: if you revoke an access token while '
                                                  'import/export jobs are running then possibly they will get wrong '
                                                  'credentials.')

    parser.set_defaults(func=revoke)
    parser.set_defaults(parser=parser)

    return parser


def revoke(args):
    fw = SdkUploadWrapper(create_flywheel_client())
    try:
        resp = fw.call_api('/gcp/auth/revoke', 'POST', response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)

    print('Successfully revoked access token')
