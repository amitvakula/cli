import sys

from flywheel.rest import ApiException

from ...config import GCPConfig
from ...sdk_impl import create_flywheel_client, SdkUploadWrapper

AVAILABLE_PROPERTIES = {
    'core': {
        'project': 'Google project id',
        'token': 'Custom Google auth token'
    },
    'healthcare': {
        'location': 'Healthcare API location (region)',
        'dataset': 'Healthcare API dataset',
        'dicomstore': 'Healthcare API dicom store'
    },
    'bigquery': {
        'dataset': 'BigQuery dataset',
        'table': 'BigQuery table'
    }
}


def add_command(subparsers):
    parser = subparsers.add_parser('set-defaults', help='Loads default config values. '
                                                        'Previously set values will be overwritten.')
    parser.set_defaults(func=load_default_config)
    parser.set_defaults(parser=parser)

    return parser


def load_default_config(args):
    fw = SdkUploadWrapper(create_flywheel_client())
    try:
        resp = fw.call_api('/gcp/config', 'GET', response_type=object)
    except ApiException as e:
        print('{} {}: {}'.format(e.status, e.reason, e.detail))
        sys.exit(1)

    config = GCPConfig()
    config.update(resp)
    config.save()

    print('Successfully set default configuration, which is the following:')

    for k, v in config.items():
        print('[{}]'.format(k))
        for kk, vv in v.items():
            print('{} = {}'.format(kk, vv))
        print('')
