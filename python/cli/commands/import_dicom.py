import re

from ..importers import DicomScanner
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

def add_command(subparsers):
    parser = subparsers.add_parser('dicom', help='Import a folder of dicom files')
    parser.add_argument('folder', help='The path to the folder to import')
    parser.add_argument('group', metavar='<id>', help='The id of the group')
    parser.add_argument('project', metavar='<label>', help='The label of the project')

    parser.add_argument('--de-identify', action='store_true', help='De-identify DICOM files, e-files and p-files prior to upload')

    parser.set_defaults(func=import_dicoms)
    parser.set_defaults(parser=parser)

    return parser

def import_dicoms(args):
    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    # Build the importer instance
    importer = DicomScanner(resolver, group=args.group, project=args.project, de_identify=args.de_identify, 
            config=args.config)

    # Perform the import
    importer.interactive_import(args.folder, resolver)


