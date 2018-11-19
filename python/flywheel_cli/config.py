import argparse
import math
import multiprocessing
import os
import zlib
import zipfile
import json

from flywheel_migration import deidentify
from .sdk_impl import create_flywheel_client, SdkUploadWrapper
from .folder_impl import FSWrapper

class Config(object):
    def __init__(self, args=None):
        self._resolver = None

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


class GCPConfig(dict):
    CONFIG_PATH = '~/.config/flywheel/gcp.json'
    AVAILABLE_PROPERTIES = {
        'core': {
            'project': 'Google project id'
        },
        'healthcare': {
            'location': 'Healthcare API location (region)',
            'dataset': 'Healthcare API dataset',
            'dicomstore': 'Healthcare API dicom store'
        },
        'bigquery': {
            'dataset': 'BigQuery dataset',
            'table': 'BigQuery table'
        }
    }

    def __init__(self):
        super(GCPConfig, self).__init__()
        self.path = os.path.expanduser(self.CONFIG_PATH)
        os.makedirs(os.path.dirname(self.path), exist_ok=True)
        self.load()

    def load(self):
        try:
            with open(self.path, 'r') as f:
                self.update(json.load(f))
        except:
            pass

    def save(self):
        with open(self.path, 'w') as f:
            json.dump(self, f)

    def update_section(self, section, payload):
        if not self.get(section):
            self[section] = {}

        self[section].update(payload)
