import argparse
import logging
import math
import multiprocessing
import os
import time
import zlib
import zipfile

from flywheel_migration import deidentify
from .sdk_impl import create_flywheel_client, SdkUploadWrapper
from .folder_impl import FSWrapper

class Config(object):
    def __init__(self, args=None):
        self._resolver = None

        # Configure logging
        self.configure_logging(args)

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

        # Assume yes option
        self.assume_yes = getattr(args, 'yes', False)
        self.max_retries = getattr(args, 'max_retries', 3)
        self.retry_wait = 5 # Wait 5 seconds between retries

        # Set output folder
        self.output_folder = getattr(args, 'output_folder', None)

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

    def get_resolver(self):
        if not self._resolver:
            if self.output_folder:
                self._resolver = FSWrapper(self.output_folder)
            else:
                fw = create_flywheel_client()
                self._resolver = SdkUploadWrapper(fw)

        return self._resolver

    def get_uploader(self):
        # Currently all resolvers are uploaders
        return self.get_resolver()

    def configure_logging(self, args):
        root = logging.getLogger()

        # Setup log format
        formatter = logging.Formatter(fmt='%(asctime)s.%(msecs)03d %(levelname)s %(message)s', datefmt='%Y-%m-%dT%H:%M:%S')
        formatter.converter = time.gmtime

        # Setup level
        if getattr(args, 'debug', False):
            root.setLevel(logging.DEBUG)
        else:
            root.setLevel(logging.INFO)

        # Initialize file logging
        log_file_path = getattr(args, 'log_file', None)
        if log_file_path:
            fileHandler = logging.FileHandler(log_file_path)
            fileHandler.setFormatter(formatter)
            root.addHandler(fileHandler)

        quiet = getattr(args, 'quiet', False)
        if not quiet:
            consoleHandler = logging.StreamHandler()
            consoleHandler.setFormatter(formatter)
            root.addHandler(consoleHandler)

    @staticmethod
    def add_deid_args(parser):
        deid_group = parser.add_mutually_exclusive_group()
        deid_group.add_argument('--de-identify', action='store_true', help='De-identify DICOM files, e-files and p-files prior to upload')
        deid_group.add_argument('--profile', help='Use the De-identify profile by name or file')

    @staticmethod
    def add_config_args(parser):
        parser.add_argument('-y', '--yes', action='store_true', help='Assume the answer is yes to all prompts')
        parser.add_argument('--max-retries', default=3, help='Maximum number of retry attempts, if assume yes')
        parser.add_argument('--jobs', '-j', default=-1, type=int, help='The number of concurrent jobs to run (e.g. compression jobs)')
        parser.add_argument('--concurrent-uploads', default=4, type=int, help='The maximum number of concurrent uploads')
        parser.add_argument('--compression-level', default=1, type=int, choices=range(-1, 9), 
                help='The compression level to use for packfiles. -1 for default, 0 for store')
        parser.add_argument('--symlinks', action='store_true', help='follow symbolic links that resolve to directories')
        parser.add_argument('--output-folder', help='Output to the given folder instead of uploading to flywheel')

        # Logging configuration
        parser.add_argument('--debug', action='store_true', help='Turn on debug logging')
        parser.add_argument('--log-file', help='Append log statements to this file')
        parser.add_argument('--quiet', action='store_true', help='Squelch log messages to the console')
