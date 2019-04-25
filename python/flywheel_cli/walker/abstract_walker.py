"""Abstract file-system walker class"""
import collections
import fnmatch

from abc import ABC, abstractmethod

class FileInfo(object):
    """Represents a node in a filesystem

    Attributes:
        name (str): The name of the file
        is_dir (bool): Whether or not the given entry is a directory
        created (datetime): When the file was created (optional)
        modified (datetime): The last time this file was modified (optional)
        size (integer): The size of the file in bytes
        is_link (bool): Whether or not this file is a symlink
    """
    def __init__(self, name, is_dir, created=None, modified=None, size=None, is_link=False):
        self.name = name
        self.is_dir = is_dir
        self.created = created
        self.modified = modified
        self.size = size
        self.is_link = is_link

    def __repr__(self):
        return 'FileInfo(name={}, is_dir={}, is_link={}, size={})'.format(
            self.name, self.is_dir, self.is_link, self.size)


class AbstractWalker(ABC):
    """Abstract interface for walking a filesystem"""
    def __init__(self, root, ignore_dot_files=True, follow_symlinks=False, filter=None, exclude=None,
            filter_dirs=None, exclude_dirs=None, delim='/'):
        """Initialize the abstract walker

        Args:
            root (str): The starting directory for walking
            ignore_dot_files (bool): Whether or not to ignore files starting with '.'
            follow_symlinks(bool): Whether or not to follow symlinks
            filter (list): An optional list of filename patterns to INCLUDE
            exclude (list): An optional list of filename patterns to EXCLUDE
            filter_dirs (list): An optional list of directories to INCLUDE
            exclude_dirs (list): An optional list of patterns of directories to EXCLUDE
            delim (str): The path delimiter, if not '/'
        """
        self.root = root

        self._ignore_dot_files = ignore_dot_files
        self._follow_symlinks = follow_symlinks

        self._include_files = filter
        self._exclude_files = exclude

        self._include_dirs = filter_dirs
        if self._include_dirs:
            self._include_dirs = [spec.split(delim) for spec in self._include_dirs]
        self._exclude_dirs = exclude_dirs

        self._delim = delim

    @abstractmethod
    def get_fs_url(self):
        """Return the FS url for the underlying filesystem"""

    def walk(self, subdir=None, max_depth=None):
        """Recursively list files in a filesystem.

        Yields:
            tuple: containing root path, a list of directories, and list of files
        """
        queue = collections.deque()
        if subdir:
            subdir = self.combine(self.root, subdir)
        else:
            subdir = self.root

        queue.append((1, subdir))

        while queue:
            # Pop next off
            depth, root = queue.popleft()

            subdirs = []
            files = []

            for item in self._listdir(root):
                full_path = self.combine(root, item.name)
                if item.is_dir:
                    if self._should_include_dir(full_path, item):
                        subdirs.append(item)

                    if max_depth is None or depth < max_depth:
                        queue.append((depth+1, full_path))
                elif self._should_include_file(full_path, item):
                    files.append(item)

            yield (root, subdirs, files)

    def files(self, subdir=None, max_depth=None):
        """Return all files in the sub directory"""
        for root, _, files in self.walk(subdir=subdir, max_depth=max_depth):
            for file_info in files:
                yield self.combine(root, file_info.name)

    @abstractmethod
    def open(path, mode='rb', **kwargs):
        """Open the given path for reading.

        Params:
            path (str): The relative or full path of the file to open
            mode (str): The open mode, either 'r' or 'rb'
            kwargs: Additional arguments to pass to open (e.g. buffering)

        Returns:
            file: a file-like object, opened for reading
        """

    @abstractmethod
    def _listdir(self, path):
        """List the contents of the given directory

        Args:
            path (str): The absolute path to the directory to list

        Yields:
            list(FileInfo): A list of file info objects
        """

    def remove_prefix(self, subdir, path):
        """Strip subdir from the beginning of path"""
        if path.startswith(subdir):
            path = path[len(subdir):]
        return path.lstrip(self._delim)

    def close(self):
        """Cleanup any resources on this walker"""

    def combine(self, part1, part2):
        """Combine two path parts with delim"""
        part1 = part1.rstrip(self._delim)
        part2 = part2.lstrip(self._delim)
        return part1 + self._delim + part2

    def _match(self, patterns, name):
        """Return true if name matches any of the given patterns"""
        for pat in patterns:
            if fnmatch.fnmatch(name, pat):
                return True
        return False

    def _should_include_dir(self, path, info):
        """Check if the given directory should be included"""
        if self._ignore_dot_files and info.name.startswith('.'):
            return False

        if not self._follow_symlinks and info.is_link:
            return False

        if self._include_dirs is not None:
            parts = (path + self._delim + info.name).lstrip(self._delim).split(self._delim)
            if not filter_match(self._include_dirs, parts):
                return False

        if self._exclude_dirs is not None and self._match(self._exclude_dirs, info.name):
            return False

        return True

    def _should_include_file(self, path, info):
        """Check if the given file should be included"""
        if self._ignore_dot_files and info.name.startswith('.'):
            return False

        if self._exclude_files is not None and self._match(self._exclude_files, info.name):
            return False

        if self._include_files is not None and not self._match(self._include_files, info.name):
            return False

        return True


### EXAMPLE CODE:
if __name__ == '__main__':
    import datetime
    import os
    import stat
    import sys

    class ConcreteWalker(AbstractWalker):
        def __init__(self, path):
            super(ConcreteWalker, self).__init__(path)

        def _listdir(self, path):
            for name in os.listdir(path):
                file_path = self.combine(path, name)
                st = os.stat(file_path)

                yield FileInfo(name, stat.S_ISDIR(st.st_mode),
                    created=datetime.datetime.fromtimestamp(st.st_ctime),
                    modified=datetime.datetime.fromtimestamp(st.st_mtime),
                    size=st.st_size, is_link=stat.S_ISLNK(st.st_mode))

        def open(self, path, mode='rb'):
            # In S3:
            # Copy to a temp folder if it doesn't already exist,
            # Then open that temp file
            if not os.path.isabs(path):
                path = os.path.join(self.root, path)

            return open(path, mode)

    walker = ConcreteWalker(sys.argv[1])
    for root, subdirs, files in walker.walk():
        print('root: {}'.format(root))
        for subdir in subdirs:
            print('  subdir: {}'.format(subdir))
        for filename in files:
            print('  file: {}'.format(filename))
