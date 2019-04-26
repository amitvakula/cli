"""Provides a scanner that will group files together under a common prefix"""
import copy

class SlurpScanner(object):
    """SlurpScanner groups files together by a common prefix.

    This works by looking at the first slash (or if there is no slash, the first dot) in
    each file path, and using that as the acquisition label.
    """
    def __init__(self, config):
        """Class that handles generic acquisition slurping"""
        self.config = config
        self.messages = []

        self.walker = config.get_walker()

    def discover(self, src_fs, context, container_factory, path_prefix=None):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            context (dict): The initial context
        """
        # Discover files first
        if path_prefix is not None:
            sub_fs = src_fs.opendir(path_prefix)
        else:
            sub_fs = src_fs

        files = list(sorted(self.walker.files(sub_fs)))

        current_prefix = None
        current_files = []

        for path in files:
            path = path.lstrip('/')

            prefix = SlurpScanner._get_prefix(path)
            if prefix == current_prefix:
                current_files.append(path)
            else:
                self._add_acquisition(container_factory, context, current_prefix, current_files)

                current_prefix = SlurpScanner._get_prefix(path)
                current_files = [path]

        self._add_acquisition(container_factory, context, current_prefix, current_files)

    @staticmethod
    def _get_prefix(path):
        """Get the appropriate prefix for the given file"""
        try:
            idx = path.rindex('/')
        except ValueError:
            try:
                idx = path.index('.')
            except ValueError:
                idx = len(path)

        return path[:idx].strip('/').replace('/', '_')

    def _add_acquisition(self, container_factory, context, label, files):
        if not label or not files:
            return

        acquisition_context = copy.deepcopy(context)
        acquisition_context.setdefault('acquisition', {})['label'] = label

        container = container_factory.resolve(acquisition_context)
        container.files.extend(files)
