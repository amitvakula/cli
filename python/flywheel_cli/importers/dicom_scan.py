import copy
import logging
import os
import sys

from .abstract_importer import AbstractImporter
from .custom_walker import CustomWalker
from ..dcm import DicomFileError, DicomFile
from .. import util

from pydicom.datadict import tag_for_keyword
from pydicom.tag import Tag

log = logging.getLogger(__name__)

class DicomSession(object):
    def __init__(self, context):
        """Helper class that holds session properties and acquisitions"""
        self.context = context
        self.acquisitions = {}

class DicomAcquisition(object):
    def __init__(self, context):
        """Helper class that holds acquisition properties and files"""
        self.context = context
        self.files = {}

# Specifying just the list of tags we're interested in speeds up dicom scanning
DICOM_TAGS = [
    'Manufacturer',
    'AcquisitionNumber',
    'AcquisitionDate',
    'AcquisitionTime',
    'SeriesDate',
    'SeriesTime',
    'SeriesInstanceUID',
    'ImageType',
    'StudyDate',
    'StudyTime',
    'StudyInstanceUID',
    'OperatorsName',
    'PatientName',
    'PatientID',
    'StudyID',
    'SeriesDescription',
    'PatientBirthDate',
    'SOPInstanceUID'
]
def _at_stack_id(tag, VR, length):
    return tag == (0x0020, 0x9056)

class DicomScanner(AbstractImporter):
    # The session label dicom header key
    session_label_key = 'StudyDescription'

    def __init__(self, resolver, group, project, config, de_identify=False, context=None): 
        """Class that handles state for dicom scanning import.

        Arguments:
            resolver (ContainerResolver): The container resolver instance
            group (str): The optional group id
            project (str): The optional project label or id in the format <id:xyz>
            config (Config): The config object
            de_identify (bool): Whether or not to de-identify DICOM, e-file, or p-file data before import. Default is False.
        """
        super(DicomScanner, self).__init__(resolver, group, project, de_identify, False, context, config)
        # Extract the following fields from dicoms:

        # session label
        # session uid
        # subject code

        # acquisition label
        # acquisition uid

        # A map of UID to DicomSession
        self.sessions = {} 

    def perform_discover(self, src_fs, context):
        """Performs discovery of containers to create and files to upload in the given folder.

        Arguments:
            src_fs (obj): The filesystem to query
            context (dict): The initial context
        """
        tags = [ Tag(tag_for_keyword(keyword)) for keyword in DICOM_TAGS ]

        # First step is to walk and sort files
        walker = CustomWalker(symlinks=self.config.follow_symlinks)
        sys.stdout.write('Scanning directories...'.ljust(80) + '\r')
        sys.stdout.flush()

        files = list(walker.files(src_fs))
        file_count = len(files)
        files_scanned = 0
        
        for path in files:
            sys.stdout.write('Scanning {}/{} files...'.format(files_scanned, file_count).ljust(80) + '\r')
            sys.stdout.flush()
            files_scanned = files_scanned+1
            
            try:
                with src_fs.open(path, 'rb', buffering=self.config.buffer_size) as f:
                    # Don't decode while scanning, stop as early as possible
                    dcm = DicomFile(f, parse=True, session_label_key=self.session_label_key, 
                        decode=False, stop_when=_at_stack_id, specific_tags=tags)
                    acquisition = self.resolve_acquisition(dcm)

                    sop_uid = dcm.get('SOPInstanceUID')
                    if sop_uid in acquisition.files:
                        orig_path = acquisition.files[sop_uid]

                        if not util.fs_files_equal(src_fs, path, orig_path):
                            log.error('File "{}" and "{}" conflict!'.format(path, orig_path))
                            log.error('Both files have the same IDs, but contents differ!')
                            exit(1)
                    else:
                        acquisition.files[sop_uid] = path

            except DicomFileError as e:
                log.info('File {} is not a dicom: {}'.format(path, e))

        sys.stdout.write(''.ljust(80) + '\n')
        sys.stdout.flush()

        # Create context objects
        for session in self.sessions.values():
            session_context = copy.deepcopy(context)
            session_context.update(session.context)

            for acquisition in session.acquisitions.values():
                acquisition_context = copy.deepcopy(session_context)
                acquisition_context.update(acquisition.context)
                files = list(acquisition.files.values())

                container = self.container_factory.resolve(acquisition_context)
                container.packfiles.append(('dicom', files, len(files)))

    def resolve_session(self, dcm):
        """Find or create a sesson from a dcm file. """
        if dcm.session_uid not in self.sessions:
            # Create session
            self.sessions[dcm.session_uid] = DicomSession({
                'session': {
                    'uid': dcm.session_uid.replace('.', ''),
                    'label': self.determine_session_label(dcm)
                },
                'subject': {
                    'label': dcm.get('PatientID', '')
                }
            })

        return self.sessions[dcm.session_uid]

    def resolve_acquisition(self, dcm):
        """Find or create an acquisition from a dcm file. """
        session = self.resolve_session(dcm)
        if dcm.series_uid not in session.acquisitions:
            session.acquisitions[dcm.series_uid] = DicomAcquisition({
                'acquisition': {
                    'uid': dcm.series_uid.replace('.', ''),
                    'label': self.determine_acquisition_label(dcm)
                }
            })

        return session.acquisitions[dcm.series_uid]

    def determine_session_label(self, dcm):
        """Determine session label from DICOM headers"""
        name = dcm.session_label
        if not name:
            if dcm.session_timestamp:
                name = dcm.session_timestamp.strftime('%Y-%m-%d %H:%M:%S')
        if not name:
            name = dcm.session_uid
        return name

    def determine_acquisition_label(self, dcm):
        """Determine acquisition label from DICOM headers"""
        name = dcm.acquisition_label
        if not name:
            if dcm.acquisition_timestamp:
                name = dcm.acquisition_timestamp.strftime('%Y-%m-%d %H:%M:%S')
        if not name:
            name = dcm.get('SeriesInstanceUID')
        return name

