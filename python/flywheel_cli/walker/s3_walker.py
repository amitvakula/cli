import boto3
import fs
import shutil
import tempfile
from urllib.parse import urlparse
from .abstract_walker import AbstractWalker, FileInfo


class S3Walker(AbstractWalker):
    """Walker that is implemented in terms of S3"""
    def __init__(self, fs_url, ignore_dot_files=True, follow_symlinks=False, filter=None, exclude=None,
                 filter_dirs=None, exclude_dirs=None, src_fs=None):
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
        schema, bucket, path, *_ = urlparse(fs_url)

        sanitized_path = '' if path == '/' else path[:-1]

        super(S3Walker, self).__init__(sanitized_path, ignore_dot_files=ignore_dot_files,
                follow_symlinks=follow_symlinks, filter=filter, exclude=exclude,
                filter_dirs=filter_dirs, exclude_dirs=exclude_dirs)
        self.bucket = bucket
        self.client = boto3.client('s3')
        self.fs_url = fs_url

    def get_fs_url(self):
        return self.fs_url

    def close(self):
        if self.dirpath:
            shutil.rmtree(self.dirpath)

    def open(self, path, mode='rb', **kwargs):
        if not self.dirpath:
            self.dirpath = tempfile.mkdtemp()

        print('We are in!')
        print('dirpath: ' + self.dirpath)
        print('path: ' + path)

        # have to see what the path looks like here
        # self.client.download_file(self.bucket, path[1:], self.dirpath + path[1:])

        # try:
            # return open(path, mode, **kwargs)
        # except fs.errors.ResourceNotFound as ex:
            # raise FileNotFoundError('File {} not found'.format(path))


    def _listdir(self, path):
        prefix_path = path[1:] if path.endswith('/') else path[1:] + '/'
        response = self.client.list_objects(Bucket=self.bucket, Prefix=prefix_path, Delimiter=self._delim)

        if 'CommonPrefixes' in response:
            common_prefixes = response['CommonPrefixes']
            for common_prefix in common_prefixes:
                prefix = common_prefix['Prefix'][:-1]
                dir_name = prefix if prefix_path == '' else prefix.split(prefix_path)[1]
                yield FileInfo(dir_name, True)

        if 'Contents' in response:
            contents = response['Contents']
            for content in contents:
                file_name = content['Key'] if prefix_path == '' else content['Key'].split(prefix_path)[1]
                last_modified = content['LastModified']
                size = content['Size']
                yield FileInfo(file_name, False, modified=last_modified, size=size)
