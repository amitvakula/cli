from abc import ABC, abstractmethod
import collections
import copy
import fs
import io
import os
import sys

from .. import util
from .container_factory import ContainerFactory
from .template import CompositeNode
from .upload_queue import SynchronousUploadQueue
from .packfile import create_zip_packfile

class AbstractImporter(ABC):
    def __init__(self, resolver, group, project, de_identify, follow_symlinks, repackage_archives, context):
        """Abstract class that handles state for flywheel imports

        Arguments:
            resolver (ContainerResolver): The container resolver instance
            group (str): The optional group id
            project (str): The optional project label or id in the format <id:xyz>
            de_identify (bool): Whether or not to de-identify DICOM, e-file, or p-file data before import. Default is False.
            follow_symlinks (bool): Whether or not to follow links (if supported by src_fs). Default is False.
            repackage_archives (bool): Whether or not to repackage (and validate and de-identify) zipped packfiles. Default is False.
            context (dict): The optional additional context fields
        """
        self.container_factory = ContainerFactory(resolver)

        self.group = group
        self.project = project
        self.de_identify = de_identify
        self.messages = []
        self.context = context
        self.follow_symlinks = follow_symlinks 
        self.repackage_archives = repackage_archives

    def initial_context(self):
        """Creates the initial context for folder import.

        Returns:
            dict: The initial context
        """
        context = {}
       
        if self.context:
            for key, value in self.context.items():
                util.set_nested_attr(context, key, value)

        if self.group:
            util.set_nested_attr(context, 'group._id', self.group)

        if self.project:
            # TODO: Check for <id:xyz> syntax
            util.set_nested_attr(context, 'project.label', self.project)

        return context

    def print_summary(self, file=sys.stdout):
        """Print a summary of the import operation in tree format.
        
        Arguments:
            file (fileobj): A file-like object that supports write(string)
        """
        # Generally - Print current container, print files, walk to next child
        spacer_str = '|   '
        entry_str = '├── '

        def write(level, msg):
            print('{}{}{}'.format(level*spacer_str, entry_str, msg), file=file)

        groups = self.container_factory.get_groups()
        queue = collections.deque([(0, group) for group in util.sorted_container_nodes(groups)])

        counts = {
            'group': 0,
            'project': 0,
            'subject': 0,
            'session': 0,
            'acquisition': 0,
            'file': 0,
            'packfile': 0
        }

        while queue:
            level, current = queue.popleft()
            cname = current.label or current.id
            status = 'using' if current.exists else 'creating'
            
            write(level, '{} ({})'.format(cname, status))

            level = level + 1
            for path in sorted(current.files, key=str.lower):
                name = fs.path.basename(path)
                write(level, fs.path.basename(path))

            for packfile_type, _, count in current.packfiles:
                write(level, '{} ({} files)'.format(packfile_type, count))

            for child in util.sorted_container_nodes(current.children):
                queue.appendleft((level, child))

            # Update counts
            counts[current.container_type] = counts[current.container_type] + 1
            counts['file'] = counts['file'] + len(current.files)
            counts['packfile'] = counts['packfile'] + len(current.packfiles)

        print('\n', file=file)
        print('This scan consists of: {} groups,'.format(counts['group']), file=file)
        print('                       {} projects,'.format(counts['project']), file=file)
        print('                       {} subjects,'.format(counts['subject']), file=file)
        print('                       {} sessions,'.format(counts['session']), file=file)
        print('                       {} acquisitions,'.format(counts['acquisition']), file=file)
        print('                       {} attachments, and'.format(counts['file']), file=file)
        print('                       {} packfiles.'.format(counts['packfile']), file=file)

    def verify(self):
        """Verify the upload plan, returning any messages that should be logged, with severity.

        Returns:
            list: A list of tuples of severity, message to be logged
        """
        results = copy.copy(self.messages)

        for _, container in self.container_factory.walk_containers():
            if container.container_type in util.NO_FILE_CONTAINERS:
                cname = container.label or container.id
                for path in container.files:
                    fname = fs.path.basename(path)
                    msg = 'File {} cannot be uploaded to {} {} - files are not supported at this level'.format(fname, container.container_type, cname)
                    results.append(('warn', msg))

                for packfile_type, _, _ in container.packfiles:
                    msg = '{} pack-file cannot be uploaded to {} {} - files are not supported at this level'.format(fname, container.container_type, cname)
                    results.append(('warn', msg))

        return results

    def discover(self, src_fs):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
        """
        context = self.initial_context()
        self.perform_discover(src_fs, context)

    @abstractmethod
    def perform_discover(src_fs, context):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            context (dict): The initial context for discovery
        """
        pass

    def interactive_import(self, folder, uploader):
        """Performs interactive import of the discovered hierarchy"""

        with fs.open_fs(util.to_fs_url(folder)) as src_fs:
            # Perform discovery on target filesystem
            self.discover(src_fs)

            # Print summary
            print('The following data hierarchy was found:\n')
            self.print_summary()

            # Print warnings
            print('')
            for severity, msg in self.verify():
                print('{} - {}'.format(severity.upper(), msg))
            print('')

            if not util.confirmation_prompt('Confirm upload?'):
                return

            # Create containers
            self.container_factory.create_containers()

            # Packfile args
            packfile_args = {
                'de_identify': self.de_identify
            }

            # Walk the hierarchy, uploading files
            upload_queue = SynchronousUploadQueue(uploader)
            for _, container in self.container_factory.walk_containers():
                cname = container.label or container.id
                packfiles = copy.copy(container.packfiles)

                for path in container.files:
                    file_name = fs.path.basename(path)

                    if self.repackage_archives and util.is_archive(path):
                        # TODO: Can we templatize or generalize this a bit?
                        with util.open_archive_fs(src_fs, path) as archive_fs:
                            if archive_fs and util.contains_dicoms(archive_fs):
                                # Do archive upload
                                packfile_data = io.BytesIO()
                                create_zip_packfile(packfile_data, archive_fs, packfile_type='dicom', symlinks=self.follow_symlinks, **packfile_args)
                                upload_queue.upload(container, file_name, packfile_data)
                                continue

                    # Normal upload
                    src = src_fs.open(path, 'rb')
                    upload_queue.upload(container, file_name, src)

                # packfiles
                for packfile_type, path, _ in container.packfiles:
                    # Don't call things foo.zip.zip
                    if packfile_type == 'zip':
                        file_name = '{}.zip'.format(cname)
                    else:
                        file_name = '{}.{}.zip'.format(cname, packfile_type)
                    
                    packfile_data = io.BytesIO()
                    if isinstance(path, str):
                        packfile_src_fs = src_fs.opendir(path)
                        create_zip_packfile(packfile_data, packfile_src_fs, packfile_type=packfile_type, symlinks=self.follow_symlinks, **packfile_args)
                    else:
                        create_zip_packfile(packfile_data, src_fs, packfile_type=packfile_type, paths=path, **packfile_args)

                    upload_queue.upload(container, file_name, packfile_data)

            upload_queue.finish()

    


