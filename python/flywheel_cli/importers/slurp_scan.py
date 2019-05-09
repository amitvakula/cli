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

    def discover(self, walker, context, container_factory, path_prefix=None):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            walker (AbstractWalker): The filesystem to query
            context (dict): The initial context
        """
        # Discover files first
        files = list(sorted(walker.files(subdir=path_prefix)))

        prefix_len = len(path_prefix or '')

        current_prefix = None
        current_files = []

        for path in files:
            path = path.lstrip('/')

            prefix = SlurpScanner._get_prefix(path[prefix_len:])
            if prefix == current_prefix:
                current_files.append(path)
            else:
                self._add_acquisition(container_factory, context, current_prefix, current_files)

                current_prefix = prefix
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
