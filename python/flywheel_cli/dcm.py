"""DICOM file wrapper"""

import collections
import os
import logging
import datetime
import string

import pydicom
import pydicom.filereader

from . import util

log = logging.getLogger(__name__)

FILETYPE = 'dicom'
GEMS_TYPE_SCREENSHOT = ['DERIVED', 'SECONDARY', 'SCREEN SAVE']
GEMS_TYPE_VXTL = ['DERIVED', 'SECONDARY', 'VXTL STATE']

class DicomFileError(pydicom.errors.InvalidDicomError):
    """DicomFileError class"""
    def __str__(self):
        """Return the wrapped exception's `str()`"""
        return str(self.args[0])

class DicomFile(object):

    """
    DicomFile class
    """

    # pylint: disable=too-few-public-methods

    def __init__(self, filepath, map_key=None, subj_key=None, session_label_key=None, parse=False, 
            de_identify=False, timezone=None, decode=True, stop_when=None, **kwargs):
        map_key = map_key.split('_', 1)[1] if map_key and map_key.startswith('RETIRED_') else map_key
        subj_key = subj_key.split('_', 1)[1] if subj_key and subj_key.startswith('RETIRED_') else subj_key
        timezone = util.DEFAULT_TZ if timezone is None else timezone
        try:
            if stop_when is not None:
                self.raw = dcm = pydicom.filereader.read_partial(filepath, stop_when, **kwargs)
            else:
                self.raw = dcm = pydicom.dcmread(filepath, stop_before_pixels=(not de_identify), **kwargs)
            if decode:
                dcm.decode()
        except (pydicom.errors.InvalidDicomError, ValueError) as ex:
            raise DicomFileError(ex)

        sort_info = dcm.get(map_key, '') if map_key else ''

        if self.get_manufacturer() != 'SIEMENS':
            self.acq_no = (str(dcm.get('AcquisitionNumber', '')) or None)
            acq_datetime = self._timestamp(dcm.get('AcquisitionDate'), dcm.get('AcquisitionTime'), timezone)
        else:
            self.acq_no = None
            acq_datetime = self._timestamp(dcm.get('SeriesDate'), dcm.get('SeriesTime'), timezone)

        if parse or de_identify:
            self.series_uid = series_uid = dcm.get('SeriesInstanceUID')
            if self._is_screenshot(dcm.get('ImageType')):
                front, back = series_uid.rsplit('.', 1)
                series_uid = front + '.' + str(int(back) - 1)
            study_datetime = self._timestamp(dcm.get('StudyDate'), dcm.get('StudyTime'), timezone)
            self.session_uid = dcm.get('StudyInstanceUID')
            self.session_label = dcm.get(session_label_key) if session_label_key else None
            self.session_timestamp = study_datetime
            self.session_operator = dcm.get('OperatorsName')
            self.subject_firstname, self.subject_lastname = self._parse_patient_name(dcm.get('PatientName', ''))
            self.subject_code, self.group__id, self.project_label = parse_sort_info(sort_info, 'ex' + dcm.get('StudyID', ''))
            if subj_key:
                self.subject_code = dcm.get(subj_key, '')
            self.acquisition_uid = series_uid + ('_' + str(self.acq_no) if self.acq_no is not None else '')
            self.acquisition_timestamp = acq_datetime or study_datetime
            self.acquisition_label = dcm.get('SeriesDescription')
            self.file_type = FILETYPE

        if de_identify:
            self.subject_firstname = self.subject_lastname = None
            if dcm.get('PatientBirthDate'):
                dob = self._parse_patient_dob(dcm.PatientBirthDate)
                if dob and study_datetime:
                    months = 12 * (study_datetime.year - dob.year) + (study_datetime.month - dob.month) - (study_datetime.day < dob.day)
                    dcm.PatientAge = '%03dM' % months if months < 960 else '%03dY' % (months / 12)
            del dcm.PatientBirthDate
            del dcm.PatientName
            del dcm.PatientID

    def save(self, dst_file):
        self.raw.save_as(dst_file)

    def get(self, key, default=None):
        return self.raw.get(key, default)

    def get_tag(self, tag_name, default=None):
        # pylint: disable=missing-docstring
        if tag_name:
            if tag_name.startswith('[') and tag_name.endswith(']'):
                tag = next((elem.tag for elem in self.raw if elem.name == tag_name), None)
                value = self.raw.get(tag).value if tag else None
            else:
                value = self.raw.get(tag_name)
            if value:
                return str(value).strip('\x00')
        return default

    def get_manufacturer(self):
        value = self.raw.get('Manufacturer')

        if not value:
            value =  ''
        elif not isinstance(value, str):
            if isinstance(value, collections.Sequence):
                value = str(value[0])
            else: # Unknown value, just convert to string
                value = str(value)

        return value.upper()

    @staticmethod
    def _is_screenshot(image_type):
        # pylint: disable=missing-docstring
        if image_type in [GEMS_TYPE_SCREENSHOT, GEMS_TYPE_VXTL]:
            return True
        return False

    @staticmethod
    def _timestamp(date, time, timezone):
        # pylint: disable=missing-docstring
        if date and time and timezone:
            return util.localize_timestamp(datetime.datetime.strptime(date + time[:6], '%Y%m%d%H%M%S'), timezone)
        return None

    @staticmethod
    def _parse_patient_name(name):
        """
        Parse patient name.

        expects "lastname" + "delimiter" + "firstname".

        Parameters
        ----------
        name : str
            string of subject first and last name, delimited by a '^' or ' '

        Returns
        -------
        firstname : str
            first name parsed from name
        lastname : str
            last name parsed from name

        """
        name = str(name)
        if '^' in name:
            lastname, _, firstname = name.partition('^')
        else:
            firstname, _, lastname = name.rpartition(' ')
        return firstname.strip().title(), lastname.strip().title()

    @staticmethod
    def _parse_patient_dob(dob):
        """
        Parse date string and sanity check.

        expects date string in YYYYMMDD format

        Parameters
        ----------
        dob : str
            dob as string YYYYMMDD

        Returns
        -------
        dob : datetime object

        """
        try:
            dob = datetime.datetime.strptime(dob, '%Y%m%d')
            if dob < datetime.datetime(1900, 1, 1):
                raise ValueError
        except (ValueError, TypeError):
            dob = None
        return dob

def parse_sort_info(sort_info, default_subject=''):
    # pylint: disable=missing-docstring
    subject, _, group_project = sort_info.strip(string.whitespace).rpartition('@')
    delimiter = next((char for char in '/:' if char in group_project), '^')
    group, _, project = group_project.partition(delimiter)
    return subject or default_subject.strip(string.whitespace), group, project

