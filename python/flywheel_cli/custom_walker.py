import fs.walk
from fnmatch import fnmatch

def filter_match(patterns, parts):
    # Fast match - assumes that if the length of parts is
    # larger than the length of the pattern, then it already matched
    # previously
    count = len(parts)
    for pattern in patterns:
        pattern_count = len(pattern)
        if count <= pattern_count:
            for i in range(count):
                if not fnmatch(parts[i], pattern[i]):
                    return False
    return True

class CustomWalker(fs.walk.Walker):
    def __init__(self, symlinks=False, ignore_dot_files=True, **kwargs):
        """A custom Walker instance that will ignore symlinks and hidden files.

        Arguments:
            symlinks (bool): Whether or not to follow symlinks (default is False)
            ignore_dot_files (bool): Whether or not to ignore hidden ('.') files (default is True)
        """
        self._symlinks = symlinks
        self._ignore_dot_files = ignore_dot_files

        # Handle filter_dirs differently than base class
        self._include_dirs = kwargs.pop('include_dirs', None) or None
        if self._include_dirs:
            self._include_dirs = [spec.split('/') for spec in self._include_dirs]

        super(CustomWalker, self).__init__(**kwargs)

    def check_file(self, fs, info):
        """Overridden to ignore dot files"""
        if self._ignore_dot_files and info.name.startswith('.'):
            return False
        return super(CustomWalker, self).check_file(fs, info)

    def check_open_dir(self, fs, path, info):
        """Overridden to ignore dot files and symlinks"""
        if self._ignore_dot_files and info.name.startswith('.'):
            return False
        if not self._symlinks:
            if info.has_namespace('link') and info.target:
                return False
        if self._include_dirs is not None:
            parts = (path + '/' + info.name).lstrip('/').split('/')
            if not filter_match(self._include_dirs, parts):
                return False
        return super(CustomWalker, self).check_open_dir(fs, path, info)

    def _iter_walk(self, fs, path, namespaces=None):
        """Overridden to add required namespaces"""
        if namespaces is None:
            namespaces = ['basic']
        if not self._symlinks:
            namespaces.append('link')
        return super(CustomWalker, self)._iter_walk(fs, path, namespaces)

