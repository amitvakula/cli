import os
import fs.path
import fs.osfs
from .importers import Uploader, ContainerResolver

class FSWrapper(Uploader, ContainerResolver):
    verb = 'Copying'

    def __init__(self, path):
        if not os.path.isdir(path):
            os.makedirs(path)

        self.dst_fs = fs.osfs.OSFS(path)

    def upload(self, container, name, fileobj):
        # Save to disk
        path = fs.path.join(container.id, name)
        if hasattr(fileobj, 'read'):
            self.dst_fs.setfile(path, fileobj)
        else:
            self.dst_fs.setbytes(path, fileobj)

    def path_el(self, container):
        if container.container_type == 'group':
            return container.id
        return container.label

    def resolve_path(self, container_type, path):
        # Resolve folder
        if self.dst_fs.exists(path):
            return path, None
        return None, None

    def create_container(self, parent, container):
        # Create folder
        if parent:
            parent_path = parent.id
            path = fs.path.join(parent_path, container.label)
        else:
            path = container.id # Group id

        self.dst_fs.makedir(path)
        return path
