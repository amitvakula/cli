import math
import multiprocessing
import zlib
import zipfile

class Config(object):
    def __init__(self, args=None):
        # Set the default compression (used by zipfile/ZipFS)
        self.compression_level = getattr(args, 'compression_level', 1) 
        if self.compression_level > 0:
            zlib.Z_DEFAULT_COMPRESSION = self.compression_level

        self.cpu_count = getattr(args, 'jobs', 1)
        if self.cpu_count == -1:
            self.cpu_count = max(1, math.floor(multiprocessing.cpu_count() / 2))

        self.concurrent_uploads = getattr(args, 'concurrent_uploads', 4)

        self.follow_symlinks = getattr(args, 'symlinks', False)

        self.buffer_size = 65536

    def get_compression_type(self):
        if self.compression_level == 0:
            return zipfile.ZIP_STORED
        return zipfile.ZIP_DEFLATED

    @staticmethod
    def add_config_args(parser):
        parser.add_argument('--jobs', '-j', default=-1, type=int, help='The number of concurrent jobs to run (e.g. compression jobs)')
        parser.add_argument('--concurrent-uploads', default=4, type=int, help='The maximum number of concurrent uploads')
        parser.add_argument('--compression-level', default=1, type=int, choices=range(-1, 9), 
                help='The compression level to use for packfiles. -1 for default, 0 for store')
        parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')



    
