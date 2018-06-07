import argparse
import fs

from ..importers import parse_template_string, FolderImporter
from .. import util
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper

from .import_folder import perform_folder_import

def add_command(subparsers):
    parser = subparsers.add_parser('template', help='Import a folder, using a template')

    parser.add_argument('--group', '-g', metavar='<id>', help='The id of the group, if not in folder structure')
    parser.add_argument('--project', '-p', metavar='<label>', help='The label of the project, if not in folder structure')

    parser.add_argument('--de-identify', action='store_true', help='De-identify DICOM files, e-files and p-files prior to upload')

    no_level_group = parser.add_mutually_exclusive_group()
    no_level_group.add_argument('--no-subjects', action='store_true', help='no subject level (create a subject for every session)')
    no_level_group.add_argument('--no-sessions', action='store_true', help='no session level (create a session for every subject)')

    parser.add_argument('--set-var', '-s', metavar='key=value', action='append', default=[], 
        type=util.split_key_value_argument, help='Set arbitrary key-value pairs')
    
    parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')

    parser.add_argument('template', help='The template string')
    parser.add_argument('folder', help='The path to the folder to import')

    parser.set_defaults(func=import_folder_with_template)
    parser.set_defaults(parser=parser)

def import_folder_with_template(args):
    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    # Build the importer instance
    importer = FolderImporter(resolver, group=args.group, project=args.project, 
        de_id=args.de_identify, merge_subject_and_session=(args.no_subjects or args.no_sessions),
        context=dict(args.set_var))

    # Build the template string
    try:
        importer.root_node = parse_template_string(args.template) 
    except (ValueError, re.error) as e:
        raise argparse.ArgumentError('Invalid template: {}'.format(e))

    # Perform the import
    with fs.open_fs(util.to_fs_url(args.folder)) as src_fs:
        perform_folder_import(resolver, importer, src_fs, args.symlinks, args.de_identify)

