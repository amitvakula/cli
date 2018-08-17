import collections
import copy
import fs
import os
import sys

from ..util import set_nested_attr, sorted_container_nodes, METADATA_ALIASES, NO_FILE_CONTAINERS
from .abstract_importer import AbstractImporter
from .container_factory import ContainerFactory
from .template import CompositeNode
from .packfile import PackfileDescriptor

class FolderImporter(AbstractImporter):
    def __init__(self, resolver, group=None, project=None,  repackage_archives=False, 
            merge_subject_and_session=False, context=None, config=None):
        """Class that handles state for folder import.

        Arguments:
            resolver (ContainerResolver): The container resolver instance
            group (str): The optional group id
            project (str): The optional project label or id in the format <id:xyz>
            repackage_archives (bool): Whether or not to repackage (and validate and de-identify) zipped packfiles. Default is False.
            merge_subject_and_session (bool): Whether or not subject or session layer is missing. Default is False.
            config (Config): The config object
        """
        super(FolderImporter, self).__init__(resolver, group, project, repackage_archives, context, config)

        self.root_node = None
        self._last_added_node = None
        self.merge_subject_and_session = merge_subject_and_session

    def add_template_node(self, next_node):
        """Append next_node to the last node that was added (or set the root node)

        Arguments:
            next_node (ImportTemplateNode): The node to append
        """
        last = self._last_added_node
        if last:
            if not hasattr(last, 'set_next'):
                raise ValueError('Cannot add node - invalid node type: {}'.format(type(last)))

            last.set_next(next_node)
        else:
            self.root_node = next_node
            
        self._last_added_node = next_node

    def add_composite_template_node(self, nodes):
        """Append a composite node to the last node that was added.

        Arguments:
            nodes (list): The list of nodes to append
        """
        composite = CompositeNode(nodes)
        self.add_template_node(composite)
        self._last_added_node = nodes[-1]

    def perform_discover(self, src_fs, context):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            follow_symlinks (bool): Whether or not to follow links (if supported by src_fs). Default is False.
            context (dict): The initial context
        """
        self.recursive_discover(src_fs, context, self.root_node, '/')

    def recursive_discover(self, src_fs, context, template_node, curdir, resolve=True):
        """Performs recursive discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            context (dict): The context object, if this is a recursive call.
            template_node (ImportTemplateNode): The current template node
            curdir (str): The current absolute path (from fs root)
            resolve (bool): Whether or not to resolve the container after discovery. Default is True.
        """
        if not context:
            context = self.initial_context()

        info_ns = ['basic']
        if not self.config.follow_symlinks:
            info_ns.append('link')

        # We only need to query for symlink if we're NOT following them
        for name in src_fs.listdir('/'):
            if name.startswith('.'):
                continue

            info = src_fs.getinfo(name, info_ns) 
            if not self.config.follow_symlinks and info.has_namespace('link') and info.target:
                continue

            path = fs.path.combine(curdir, name)

            if info.is_dir:
                next_node = None
                with src_fs.opendir(name) as subdir:
                    if 'packfile' in context:
                        child_context = context
                    else:
                        child_context = context.copy()
                        child_context.pop('files', None)

                        if template_node is None:
                            # Treat as packfile
                            child_context['packfile'] = name
                        else:
                            next_node = template_node.extract_metadata(name, child_context, src_fs)

                    resolve_child = 'packfile' not in context
                    self.recursive_discover(subdir, child_context, next_node, path, resolve=resolve_child)
            else:
                context.setdefault('files', []).append(path)

        # Resolve the container
        if resolve:
            container = self.container_factory.resolve(context)

            files = context.get('files', None)
            if files:
                if container:
                    if 'packfile' in context:
                        packfile_name = context.get('packfile_name')
                        container.packfiles.append(PackfileDescriptor(
                            context['packfile'], curdir, len(files), name=packfile_name
                        ))
                    else:
                        container.files.extend(files)
                else:
                    self.messages.append(('warn', 'Ignoring files for folder {} because it represents an ambiguous node'.format(curdir)))

