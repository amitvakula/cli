import copy
import io
import re

import fs

from ..importers import (
    create_zip_packfile, 
    ContainerFactory, 
    FolderImporter, 
    StringMatchNode,
    SynchronousUploadQueue
)

from ..sdk_impl import create_flywheel_client, SdkUploadWrapper
from .. import util

def add_command(subparsers):
    parser = subparsers.add_parser('folder', help='Import a structured folder')
    parser.add_argument('folder', help='The path to the folder to import')
    parser.add_argument('--group', '-g', metavar='<id>', help='The id of the group, if not in folder structure')
    parser.add_argument('--project', '-p', metavar='<label>', help='The label of the project, if not in folder structure')

    # Cannot specify dicom folder name with dicom-acquistions, or bruker-acquisitions with either
    acq_group = parser.add_mutually_exclusive_group()
    acq_group.add_argument('--dicom', default='dicom', metavar='name', help='The name of dicom subfolders to be zipped prior to upload')
    acq_group.add_argument('--pack-acquisitions', metavar='type', help='Acquisition folders only contain acquisitions of <type> and are zipped prior to upload')
    
    parser.add_argument('--de-identify', action='store_true', help='De-identify DICOM files, e-files and p-files prior to upload')

    no_level_group = parser.add_mutually_exclusive_group()
    no_level_group.add_argument('--no-subjects', action='store_true', help='no subject level (create a subject for every session)')
    no_level_group.add_argument('--no-sessions', action='store_true', help='no session level (create a session for every subject)')

    parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')
    parser.add_argument('--root-dirs', type=int, default=0, help='The number of directories to discard before matching')

    parser.set_defaults(func=import_folder)
    parser.set_defaults(parser=parser)

    return parser

def import_folder(args):
    # Validate that if project is set, then group is set
    if args.project and not args.group:
        args.parser.error('Specifying project requires also specifying group')

    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    # Build the importer instance
    importer = FolderImporter(resolver, group=args.group, project=args.project, 
        de_id=args.de_identify, merge_subject_and_session=(args.no_subjects or args.no_sessions))

    for i in range(args.root_dirs):
        importer.add_template_node(StringMatchNode(re.compile('.*'))) 

    if not args.group:
        importer.add_template_node(StringMatchNode('group'))

    if not args.project:
        importer.add_template_node(StringMatchNode('project'))

    if not args.no_subjects:
        importer.add_template_node(StringMatchNode('subject'))

    if not args.no_sessions:
        importer.add_template_node(StringMatchNode('session'))

    if args.pack_acquisitions:
        importer.add_template_node(StringMatchNode('acquisition', packfile_type=args.pack_acquisitions))
    else:
        importer.add_template_node(StringMatchNode('acquisition'))
        importer.add_template_node(StringMatchNode(re.compile(args.dicom), packfile_type='dicom'))

    # Perform the import
    with fs.open_fs(util.to_fs_url(args.folder)) as src_fs:
        perform_folder_import(resolver, importer, src_fs, args.symlinks, args.de_identify)

def perform_folder_import(resolver, importer, src_fs, symlinks, de_identify, repackage_archives=False):
    # Perform discovery on target filesystem
    importer.discover(src_fs, symlinks)

    # Print summary
    print('The following data hierarchy was found:\n')
    importer.print_summary()

    # Print warnings
    print('')
    for severity, msg in importer.verify():
        print('{} - {}'.format(severity.upper(), msg))
    print('')

    if not util.confirmation_prompt('Confirm upload?'):
        return

    # Create containers
    importer.container_factory.create_containers()

    # Packfile args
    packfile_args = {
        'de_identify': de_identify
    }

    # Walk the hierarchy, uploading files
    upload_queue = SynchronousUploadQueue(resolver)
    for _, container in importer.container_factory.walk_containers():
        cname = container.label or container.id
        packfiles = copy.copy(container.packfiles)

        for path in container.files:
            file_name = fs.path.basename(path)

            if repackage_archives and util.is_archive(path):
                # TODO: Can we templatize or generalize this a bit?
                with util.open_archive_fs(src_fs, path) as archive_fs:
                    if archive_fs and util.contains_dicoms(archive_fs):
                        # Do archive upload
                        packfile_data = io.BytesIO()
                        packfile = create_zip_packfile(packfile_data, archive_fs, 'dicom', **packfile_args)
                        upload_queue.upload(container, file_name, packfile_data)
                        continue

            # Normal upload
            src = src_fs.open(path, 'rb')
            upload_queue.upload(container, file_name, src)

        # packfiles
        for packfile_type, path, _ in container.packfiles:
            file_name = '{}.{}.zip'.format(cname, packfile_type)
            
            packfile_data = io.BytesIO()
            packfile_src_fs = src_fs.opendir(path)
            packfile = create_zip_packfile(packfile_data, packfile_src_fs, packfile_type, **packfile_args)
            upload_queue.upload(container, file_name, packfile_data)

    upload_queue.finish()


