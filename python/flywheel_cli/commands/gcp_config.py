import io
import sys
import textwrap
from argparse import RawTextHelpFormatter

from flywheel.rest import ApiException

from ..config import GCPConfig
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

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


def gen_available_properties_doc():
    doc = io.StringIO()
    doc.write('AVAILABLE PROPERTIES:\n')
    for k, v in AVAILABLE_PROPERTIES.items():
        doc.write('  {}\n'.format(k))
        for kk, vv in v.items():
            doc.write('    {}\n'.format(kk))
            doc.write('      {}\n'.format(vv))
    doc.seek(0)
    return doc.read()


def add_command(subparsers):
    parser = subparsers.add_parser('config', help='The fw gcp config command group lets you set, '
                                                  'view and unset properties used by fw commands '
                                                  'which are connected to Google Cloud')
    subsubparsers = parser.add_subparsers(title='Available commands', metavar='')

    set_parser = subsubparsers.add_parser('set',
                                          help='Set a property',
                                          epilog=gen_available_properties_doc(),
                                          formatter_class=RawTextHelpFormatter)
    set_parser.add_argument('property_name', metavar='[SECTION].PROPERTY',
                            help=textwrap.dedent('''\
                            Property to be set. Section is optional while
                            referring to core properties, so using either core.project or
                            project is a valid way of setting the project, while using
                            section names is essential for setting specific properties like
                            bq.dataset. See the available properties section.
                            '''))
    set_parser.add_argument('value', help='Value to be set.', metavar='VALUE')

    set_parser.set_defaults(func=set_config)
    set_parser.set_defaults(parser=set_parser)

    list_parser = subsubparsers.add_parser('list', help='List properties')
    list_parser.set_defaults(func=list_config)
    list_parser.set_defaults(parser=list_parser)

    unset_parser = subsubparsers.add_parser('unset', help='Unset a property')
    unset_parser.add_argument('property_name', metavar='[SECTION].PROPERTY',
                              help='Property to be set. Section is optional while '
                                   'referring to core properties')
    unset_parser.set_defaults(func=unset_config)
    unset_parser.set_defaults(parser=unset_parser)

    load_default_parser = subsubparsers.add_parser('set-defaults', help='Loads default config values. '
                                                                        'Previously set values will be overwritten.')
    load_default_parser.set_defaults(func=load_default_config)
    load_default_parser.set_defaults(parser=load_default_parser)

    return parser


def set_config(args):

    config = GCPConfig()

    prop_key_parts = args.property_name.split('.')

    section = 'core'

    if len(prop_key_parts) == 2:
        section = prop_key_parts[0]
        prop_name = prop_key_parts[1]
    else:
        prop_name = prop_key_parts[0]

    payload = {
        prop_name: args.value
    }

    if section not in AVAILABLE_PROPERTIES:
        print('Invalid section')
        sys.exit(1)

    if prop_name not in AVAILABLE_PROPERTIES[section]:
        print('Invalid property')
        sys.exit(1)

    print('Updating GCP config...')
    config.update_section(section, payload)
    config.save()

    print('Successfully set [{}.{}] to [{}]'.format(section, prop_name, args.value))


def list_config(args):
    config = GCPConfig()

    for k, v in config.items():
        print('[{}]'.format(k))
        for kk, vv in v.items():
            print('{} = {}'.format(kk, vv))
        print('')


def unset_config(args):
    config = GCPConfig()
    prop_key_parts = args.property_name.split('.')

    section = 'core'

    if len(prop_key_parts) == 2:
        section = prop_key_parts[0]
        prop_name = prop_key_parts[1]
    else:
        prop_name = prop_key_parts[0]

    removed = config[section].pop(prop_name)

    print('Successfully set [{}.{}]. Value was [{}]'.format(section, prop_name, removed))


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

