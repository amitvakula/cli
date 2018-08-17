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
from .upload_queue import UploadQueue
from .packfile import create_zip_packfile

class AbstractImporter(ABC):
    # Whether or not archive filesystems are supported
    support_archive_fs = True

    def __init__(self, resolver, group, project, repackage_archives, context, config):
        """Abstract class that handles state for flywheel imports

        Arguments:
            resolver (ContainerResolver): The container resolver instance
            group (str): The optional group id
            project (str): The optional project label or id in the format <id:xyz>
            repackage_archives (bool): Whether or not to repackage (and validate and de-identify) zipped packfiles. Default is False.
            context (dict): The optional additional context fields
            config (Config): The config object
        """
        self.container_factory = ContainerFactory(resolver)

        self.group = group
        self.project = project
        self.messages = []
        self.context = context
        self.config = config
        self.repackage_archives = repackage_archives

        if config:
            self.deid_profile = config.deid_profile
        else:
            self.deid_profile = None

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

            for desc in current.packfiles:
                label = desc.name if desc.name else desc.packfile_type
                write(level, '{} ({} files)'.format(label, desc.count))

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

        return counts

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

                for desc in container.packfiles:
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
        try:
            fs_url = util.to_fs_url(folder, self.support_archive_fs)
        except util.UnsupportedFilesystemError as e:
            print(e)
            return

        with fs.open_fs(fs_url) as src_fs:
            # Perform discovery on target filesystem
            self.discover(src_fs)

            if self.container_factory.is_empty():
                print('Nothing found to import!')
                return

            # Print summary
            print('The following data hierarchy was found:\n')
            counts = self.print_summary()

            # Print warnings
            print('')
            for severity, msg in self.verify():
                print('{} - {}'.format(severity.upper(), msg))
            print('')

            if not util.confirmation_prompt('Confirm upload?'):
                return

            # Create containers
            self.container_factory.create_containers()

            # Walk the hierarchy, uploading files
            upload_queue = UploadQueue(uploader, self.config, upload_count=counts['file'], packfile_count=counts['packfile'])
            upload_queue.start()

            for _, container in self.container_factory.walk_containers():
                cname = container.label or container.id
                packfiles = copy.copy(container.packfiles)

                for path in container.files:
                    file_name = fs.path.basename(path)

                    if self.repackage_archives and util.is_archive(path):
                        archive_fs = util.open_archive_fs(src_fs, path)
                        if archive_fs:
                            if util.contains_dicoms(archive_fs):
                                # Repackage upload
                                upload_queue.upload_packfile(archive_fs, 'dicom', self.deid_profile, container, file_name)
                                continue
                            else:
                                archive_fs.close()

                    # Normal upload
                    src = src_fs.open(path, 'rb')
                    # TODO: upload_queue.upload_file()
                    upload_queue.upload(container, file_name, src)

                # packfiles
                for desc in container.packfiles:
                    if desc.name:
                        file_name = desc.name
                    else:
                        # Don't call things foo.zip.zip
                        packfile_name = util.str_to_filename(cname)
                        if desc.packfile_type == 'zip':
                            file_name = '{}.zip'.format(packfile_name)
                        else:
                            file_name = '{}.{}.zip'.format(packfile_name, desc.packfile_type)
                    
                    if isinstance(desc.path, str):
                        packfile_src_fs = src_fs.opendir(desc.path)
                        upload_queue.upload_packfile(packfile_src_fs, desc.packfile_type, self.deid_profile, container, file_name)
                    else:
                        packfile_src_fs = src_fs.opendir('/')
                        upload_queue.upload_packfile(src_fs, desc.packfile_type, self.deid_profile, container, file_name, paths=desc.path)

            upload_queue.wait_for_finish()
            # Retry loop for errored jobs
            while upload_queue.has_errors():

                upload_queue.suspend_reporting()
                print('')
                if not util.confirmation_prompt('One or more errors occurred. Retry?'):
                    break

                # Requeue and wait for finish
                upload_queue.requeue_errors()
                upload_queue.resume_reporting()
                upload_queue.wait_for_finish()

            upload_queue.shutdown()

