import io
import sys
import textwrap
from argparse import RawTextHelpFormatter

from ...config import GCPConfig


def gen_available_properties_doc():
    doc = io.StringIO()
    doc.write('AVAILABLE PROPERTIES:\n')
    for k, v in GCPConfig.AVAILABLE_PROPERTIES.items():
        doc.write('  {}\n'.format(k))
        for kk, vv in v.items():
            doc.write('    {}\n'.format(kk))
            doc.write('      {}\n'.format(vv))
    doc.seek(0)
    return doc.read()


def add_command(subparsers):
    parser = subparsers.add_parser('set',
                                   help='Set a property',
                                   epilog=gen_available_properties_doc(),
                                   formatter_class=RawTextHelpFormatter)
    parser.add_argument('property_name', metavar='[SECTION].PROPERTY',
                        help=textwrap.dedent('''\
                        Property to be set. Section is optional while
                        referring to core properties, so using either core.project or
                        project is a valid way of setting the project, while using
                        section names is essential for setting specific properties like
                        bq.dataset. See the available properties section.
                        '''))
    parser.add_argument('value', help='Value to be set.', metavar='VALUE')

    parser.set_defaults(func=set_config)
    parser.set_defaults(parser=parser)

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

    if section not in GCPConfig.AVAILABLE_PROPERTIES:
        print('Invalid section')
        sys.exit(1)

    if prop_name not in GCPConfig.AVAILABLE_PROPERTIES[section]:
        print('Invalid property')
        sys.exit(1)

    print('Updating GCP config...')
    config.update_section(section, payload)
    config.save()

    print('Successfully set [{}.{}] to [{}]'.format(section, prop_name, args.value))
