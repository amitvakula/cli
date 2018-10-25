import argparse
import logging
import logging.handlers
import math
import multiprocessing
import os
import time
import zlib
import zipfile

from flywheel_migration import deidentify
from .sdk_impl import create_flywheel_client, SdkUploadWrapper
from .folder_impl import FSWrapper

CLI_LOG_MAX_BYTES = 5242880 # 5MB
CLI_LOG_PATH = '~/.cache/flywheel/logs/cli.log'

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

        # Propagate all debug logging
        root.setLevel(logging.DEBUG)

        # Always log to cli log file
        log_path = os.path.expanduser(CLI_LOG_PATH)
        log_dir = os.path.dirname(log_path)
        if not os.path.isdir(log_dir):
            os.makedirs(log_dir)

        # Use GMT ISO date for logfile
        file_formatter = logging.Formatter(fmt='%(asctime)s.%(msecs)03d %(levelname)s %(message)s', datefmt='%Y-%m-%dT%H:%M:%S')
        file_formatter.converter = time.gmtime

        file_handler = logging.handlers.RotatingFileHandler(log_path, maxBytes=CLI_LOG_MAX_BYTES, backupCount=2)
        file_handler.setFormatter(file_formatter)
        root.addHandler(file_handler)

        # Control how much (if anything) goes to console
        console_log_level = logging.INFO
        if getattr(args, 'quiet', False):
            console_log_level = logging.ERROR
        elif getattr(args, 'debug', False):
            console_log_level = logging.DEBUG

        console_formatter = logging.Formatter(fmt='%(levelname)s: %(message)s')

        console_handler = logging.StreamHandler()
        console_handler.setFormatter(console_formatter)
        console_handler.setLevel(console_log_level)
        root.addHandler(console_handler)

        # Finally, capture all warnings to the logging framework
        logging.captureWarnings(True)

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
        log_group = parser.add_mutually_exclusive_group()
        log_group.add_argument('--debug', action='store_true', help='Turn on debug logging')
        log_group.add_argument('--quiet', action='store_true', help='Squelch log messages to the console')
