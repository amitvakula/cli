import copy
import fs
import logging
import sys
import datetime

log = logging.getLogger(__name__)

from .abstract_importer import AbstractImporter
from .packfile import PackfileDescriptor
from .template import parse_template_string
from .. import util

# TODO use external implementation once available
# from flywheel_migration.pet_ct_siemens import parse_siemens_hdr

def parse_siemens_hdr(header_file):
    # pylint: disable=missing-docstring
    header = {}
    for line in header_file:
        if line.startswith('gmt_scan_time'):
            timestamp = datetime.datetime.strptime(line, 'gmt_scan_time %a %b %d %H:%M:%S %Y\n')
            header['Scan start date and time - GMT-based'] = timestamp 
            break
    return header


class PetCtAcquisition(object):
    def __init__(self, context):
        """Helper class that holds acquisition properties and files"""
        self.context = context
        self.hdr_file = None


class PetCtSession(object):
    def __init__(self, context):
        """Helper class that holds session properties and acquisitions"""
        self.context = context
        self.acquisitions = {}


class PetCtScanner(object):


    def __init__(self, config):
        
        self.config = config
        self.sessions = {}
        self.messages = []
        self.template = None


    def discover(self, src_fs, context, container_factory, path_prefix=None):

        sys.stdout.write('Scanning directories...'.ljust(80) + '\r')
        sys.stdout.flush()
            
        hdr_paths = [fp.path for fp in src_fs.glob('/**/*.img.hdr')]
        files_scanned = 0
        file_count = len(hdr_paths)

        for hdr_path in hdr_paths:
            sys.stdout.write('Scanning {}/{} .hdr files...'.format(files_scanned, file_count).ljust(80) + '\r')
            sys.stdout.flush()
            files_scanned = files_scanned + 1 

            real_path = path_prefix + hdr_path if path_prefix else hdr_path

            try:
                with src_fs.open(real_path, 'r') as f:
                    hdr_meta = parse_siemens_hdr(f)

                acquisition = self.resolve_acquisition(context, hdr_meta, real_path)
                acquisition.hdr_file = real_path
           
            except Exception as e:
                log.exception('Unable parse HDR file: %s', real_path)

        sys.stdout.write(''.ljust(80) + '\n')
        sys.stdout.flush()

        for session in self.sessions.values():
            session_context = copy.deepcopy(context)
            session_context.update(session.context)

            for acquisition in session.acquisitions.values():
                acquisition_context = copy.deepcopy(session_context)
                acquisition_context.update(acquisition.context)

                imported_files = [acquisition.hdr_file]
                no_ext_path = acquisition.hdr_file.replace('.img.hdr', '')

                img_file = no_ext_path + '.img'
                if src_fs.isfile(img_file):
                    imported_files.append(img_file)
                else:
                    log.warning('Not found .img file for %s file', acquisition.hdr_file)

                log_file = no_ext_path + '.log'
                if src_fs.isfile(log_file):
                    imported_files.append(log_file)
                else:
                    log.warning('Not found .log file for %s file', acquisition.hdr_file)

                container = container_factory.resolve(acquisition_context)
                container.packfiles.append(PackfileDescriptor('pet-ct', imported_files, len(imported_files)))


    def resolve_session(self, context, hdr_meta, hdr_path):

        if self.template:
            self.template.extract_metadata(hdr_path, context)

        session_label = context.get('session', {}).get('label')
        subject_label = context.get('subject', {}).get('label', session_label)

        session_key = (subject_label, session_label)
        timestamp = util.localize_timestamp(hdr_meta['Scan start date and time - GMT-based'])

        if session_key not in self.sessions:
            self.sessions[session_key] = PetCtSession({
                    'session': {
                        'label': session_label,
                        'timestamp': timestamp,
                        'timezone': str(util.DEFAULT_TZ)
                    },
                    'subject': {
                        'label': subject_label
                    }
                })

        return self.sessions[session_key]


    def resolve_acquisition(self, context, hdr_meta, hdr_path):

        hdr_path, hdr_file_name = hdr_path.rsplit('/', 1)

        session = self.resolve_session(context, hdr_meta, hdr_path)
        session.acquisitions[hdr_file_name] = PetCtAcquisition({
                'acquisition': {
                    'label': hdr_file_name.replace('.img.hdr', '')
                }
        })
        return session.acquisitions[hdr_file_name]


class PetCtScannerImporter(AbstractImporter):


    def __init__(self, group, project, config, context=None, subject_label=None, session_label=None, template=None,
                 zip_path_transform=None):

        super(PetCtScannerImporter, self).__init__(group, project, True, context, config, zip_path_transform)

        self.scanner = PetCtScanner(config)
        self.subject_label = subject_label
        self.session_label = session_label


    def add_template(self, template):
        self.scanner.template = parse_template_string(template)


    def initial_context(self):
        """Creates the initial context for folder import.

        Returns:
            dict: The initial context
        """
        context = super(PetCtScannerImporter, self).initial_context()

        if self.subject_label:
            util.set_nested_attr(context, 'subject.label', self.subject_label)

        if self.session_label:
            util.set_nested_attr(context, 'session.label', self.session_label)

        return context


    def perform_discover(self, src_fs, context):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            context (dict): The initial context
        """
        self.scanner.discover(src_fs, context, self.container_factory)
        self.messages += self.scanner.messages
