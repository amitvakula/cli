import collections
import copy
import fs
import os
import sys

from ..util import set_nested_attr, sorted_container_nodes, METADATA_ALIASES, NO_FILE_CONTAINERS
from .container_factory import ContainerFactory

class FolderImporter(object):
    def __init__(self, resolver, group=None, project=None, de_id=False, merge_subject_and_session=False):
        """Class that handles state for folder import.

        Arguments:
            resolver (ContainerResolver): The container resolver instance
            group (str): The optional group id
            project (str): The optional project label or id in the format <id:xyz>
            de_id (bool): Whether or not to de-identify DICOM, e-file, or p-file data before import. Default is False.
            merge_subject_and_session (bool): Whether or not subject or session layer is missing. Default is False.
        """
        self.root_node = None
        self.container_factory = ContainerFactory(resolver)
        self.group = group
        self.project = project
        self.de_id = de_id
        self.merge_subject_and_session = merge_subject_and_session
        self.messages = []

    def initial_context(self):
        """Creates the initial context for folder import.

        Returns:
            dict: The initial context
        """
        context = {}
        
        if self.group:
            set_nested_attr(context, 'group._id', self.group)

        if self.project:
            # TODO: Check for <id:xyz> syntax
            set_nested_attr(context, 'project.label', self.project)

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
        queue = collections.deque([(0, group) for group in sorted_container_nodes(groups)])

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

            for child in sorted_container_nodes(current.children):
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
            if container.container_type in NO_FILE_CONTAINERS:
                cname = container.label or container.id
                for path in container.files:
                    fname = fs.path.basename(path)
                    msg = 'File {} cannot be uploaded to {} {} - files are not supported at this level'.format(fname, container.container_type, cname)
                    results.append(('warn', msg))

                for packfile_type, _, _ in container.packfiles:
                    msg = '{} pack-file cannot be uploaded to {} {} - files are not supported at this level'.format(fname, container.container_type, cname)
                    results.append(('warn', msg))

        return results

    def discover(self, src_fs, follow_symlinks=False):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            follow_symlinks (bool): Whether or not to follow links (if supported by src_fs). Default is False.
        """
        context = self.initial_context()
        self.recursive_discover(src_fs, follow_symlinks, context, self.root_node, '/')

    def recursive_discover(self, src_fs, follow_symlinks, context, template_node, curdir):
        """Performs recursive discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            follow_symlinks (bool): Whether or not to follow links (if supported by src_fs). Default is False.
            context (dict): The context object, if this is a recursive call.
            template_node (ImportTemplateNode): The current template node
            curdir (str): The current absolute path (from fs root)
        """
        if not context:
            context = self.initial_context()

        # We only need to query for symlink if we're NOT following them
        info_ns = ['basic']
        if not follow_symlinks:
            info_ns.append('link')

        for name in src_fs.listdir('/'):
            if name.startswith('.'):
                continue

            info = src_fs.getinfo(name, info_ns) 
            if not follow_symlinks and info.has_namespace('link') and info.is_link:
                continue

            path = fs.path.combine(curdir, name)
            if info.is_dir:
                context_copy = context.copy()
                context_copy.pop('files', None)

                with src_fs.opendir(name) as subdir:
                    if template_node is None:
                        # Treat as packfile
                        context_copy['packfile'] = name
                    else:
                        next_node = template_node.extract_metadata(name, context_copy, subdir)

                    self.recursive_discover(subdir, follow_symlinks, context_copy, next_node, path)
            else:
                context.setdefault('files', [])
                context['files'].append(path)

        # Resolve the container
        container = self.container_factory.resolve(context)

        files = context.get('files', None)
        if files:
            if container:
                if 'packfile' in context:
                    container.packfiles.append((context['packfile'], curdir, len(files)))
                else:
                    container.files.extend(files)
            else:
                self.messages.append(('warn', 'Ignoring files for folder {} because it represents an ambiguous node'.format(curdir)))

