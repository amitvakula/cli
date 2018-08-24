import argparse
import math
import multiprocessing
import os
import zlib
import zipfile

from flywheel_migration import deidentify

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

        # Get de-identification profile
        if getattr(args, 'de_identify', False):
            profile_name = 'minimal'
        else:
            profile_name = getattr(args, 'profile', None)

        if not profile_name:
            profile_name = 'none'

        self.deid_profile = self.load_deid_profile(profile_name, args=args)

    def get_compression_type(self):
        if self.compression_level == 0:
            return zipfile.ZIP_STORED
        return zipfile.ZIP_DEFLATED

    def load_deid_profile(self, name, args=None):
        if os.path.isfile(name):
            return deidentify.load_profile(name)

        # Load default profiles
        profiles = deidentify.load_default_profiles()
        for profile in profiles:
            if profile.name == name:
                return profile

        msg = 'Unknown de-identification profile: {}'.format(name)
        if args:
            args.parser.error(msg)
        else:
            raise ValueError(msg)

    @staticmethod
    def add_deid_args(parser):
        deid_group = parser.add_mutually_exclusive_group()
        deid_group.add_argument('--de-identify', action='store_true', help='De-identify DICOM files, e-files and p-files prior to upload')
        deid_group.add_argument('--profile', help='Use the De-identify profile by name or file')

    @staticmethod
    def add_config_args(parser):
        parser.add_argument('--jobs', '-j', default=-1, type=int, help='The number of concurrent jobs to run (e.g. compression jobs)')
        parser.add_argument('--concurrent-uploads', default=4, type=int, help='The maximum number of concurrent uploads')
        parser.add_argument('--compression-level', default=1, type=int, choices=range(-1, 9), 
                help='The compression level to use for packfiles. -1 for default, 0 for store')
        parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')