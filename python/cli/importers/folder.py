import copy
import fs
import os
from ..util import set_nested_attr, METADATA_ALIASES
from .container_factory import ContainerFactory

class ImportFolderNode(object):
    def __init__(self, metavar=None, name=None, packfile_type=None):
        """Represents a templated path node for folder import.

        Arguments:
            metavar (str): What variable this node matches (may be one of METADATA_ALIASES)
            name (str): Alternatively, what the folder should be named
            packfile_type (str): What packfile type this node represents
        """
        self.metavar = metavar
        self.name = name
        self.packfile_type = packfile_type

    def extract_metadata(self, name, context):
        """Extracts metadata from a path element, into context.

        Arguments:
            name (str): The current path element name (i.e. folder name)
            context (dict): The context object to update
        """
        if self.name:
            if name == self.name:
                if self.packfile_type:
                    context['packfile'] = self.packfile_type
            else:
                context['packfile'] = name

        if self.metavar:
            if self.metavar in METADATA_ALIASES:
                key = METADATA_ALIASES[self.metavar]
            else:
                key = self.metavar

            set_nested_attr(context, key, name)

            if self.packfile_type:
                context['packfile'] = self.packfile_type

    def __str__(self):
        """String representation of this node"""
        if self.metavar:
            packfile_str = ''
            if self.packfile_type:
                packfile_str = ',type={}'.format(self.packfile_type)
            return '{{{0}{1}}}'.format(self.metavar, packfile_str)
        if self.name:
            return self.name
        return '*'

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
        self.template = []
        self.container_factory = ContainerFactory(resolver)
        self.group = group
        self.project = project
        self.de_id = de_id
        self.merge_subject_and_session = merge_subject_and_session

    def add_template_node(self, **kwargs):
        """Adds a template node to the folder path
        
        Arguments:
            **kwargs: See arguments for ImportFolderNode.
        """
        self.template.append(ImportFolderNode(**kwargs))

    def get_template_str(self):
        """Returns a string representation of the folder template.

        Returns:
            str: The string representation of the folder template
        """
        result = ''
        for node in self.template:
            if result:
                result = os.path.join(result, str(node))
            else:
                result = str(node)
        return result

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

    def discover(self, src_fs, follow_symlinks=False, context=None, template_idx=0, curdir='/'):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            follow_symlinks (bool): Whether or not to follow links (if supported by src_fs). Default is False.
            context (dict): The context object, if this is a recursive call.
            template_idx (int): The current index in the template path
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

                if template_idx >= len(self.template):
                    # Treat as packfile
                    context_copy['packfile'] = name
                else:
                    import_node = self.template[template_idx]
                    import_node.extract_metadata(name, context_copy)

                with src_fs.opendir(name) as subdir:
                    self.discover(subdir, follow_symlinks=follow_symlinks, context=context_copy, template_idx=template_idx+1, curdir=path)

            else:
                context.setdefault('files', [])
                context['files'].append(path)

        # Resolve the container
        container = self.container_factory.resolve(context)

        files = context.get('files', None)
        if files:
            if container:
                if 'packfile' in context:
                    container.packfiles.append((context['packfile'], files))
                else:
                    container.files.extend(files)
            else:
                # TODO: log
                print('Ignoring files for folder {} because it represents an ambiguous node'.format(curdir))




