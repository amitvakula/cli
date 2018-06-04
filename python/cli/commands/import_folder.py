import fs
from ..importers import FolderImporter, ContainerFactory, SynchronousUploadQueue
from ..sdk_impl import create_flywheel_client, SdkUploadWrapper
from ..util import to_fs_url, confirmation_prompt

def add_command(subparsers):
    parser = subparsers.add_parser('folder', help='Import a structured folder')
    parser.add_argument('folder', help='The path to the folder to import')
    parser.add_argument('--group', '-g', metavar='<id>', help='The id of the group, if not in folder structure')
    parser.add_argument('--project', '-p', metavar='<label>', help='The label of the project, if not in folder structure')

    # Cannot specify dicom folder name with dicom-acquistions, or bruker-acquisitions with either
    acq_group = parser.add_mutually_exclusive_group()
    acq_group.add_argument('--dicom', default='dicom', metavar='name', help='The name of dicom subfolders to be zipped prior to upload')
    acq_group.add_argument('--pack-acquisitions', metavar='type', help='Acquisition folders only contain acquisitions of <type> and are zipped prior to upload')
    
    parser.add_argument('--de-id', action='store_true', help='De-identify DICOM files, e-files and p-files prior to upload')

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

    resolver, importer = build_folder_importer(args)
    print('Template: {}'.format(importer.get_template_str()))

    src_fs = fs.open_fs(to_fs_url(args.folder))

    importer.discover(src_fs, args.symlinks)

    # Print summary
    print('The following data hierarchy was found:\n')
    importer.print_summary()

    # Print warnings
    print('')
    for severity, msg in importer.verify():
        print('{} - {}'.format(severity.upper(), msg))
    print('')

    if not confirmation_prompt('Confirm upload?'):
        return

    # Create containers
    importer.container_factory.create_containers()

    # Walk the hierarchy, uploading files
    upload_queue = SynchronousUploadQueue(resolver)
    for _, container in importer.container_factory.walk_containers():
        for path in container.files:
            src = src_fs.open(path, 'rb')
            file_name = fs.path.basename(path)

            upload_queue.upload(container, file_name, src)


def build_folder_importer(args):
    fw = create_flywheel_client()
    resolver = SdkUploadWrapper(fw)

    importer = FolderImporter(resolver, group=args.group, project=args.project, 
        de_id=args.de_id, merge_subject_and_session=(args.no_subjects or args.no_sessions))

    for i in range(args.root_dirs):
        importer.add_template_node()

    if not args.group:
        importer.add_template_node(metavar='group')

    if not args.project:
        importer.add_template_node(metavar='project')

    if not args.no_subjects:
        importer.add_template_node(metavar='subject')

    if not args.no_sessions:
        importer.add_template_node(metavar='session')

    if args.pack_acquisitions:
        importer.add_template_node(metavar='acquisition', packfile_type=args.pack_acquisitions)
    else:
        importer.add_template_node(metavar='acquisition')
        importer.add_template_node(name=args.dicom, packfile_type='dicom')

    return resolver, importer

