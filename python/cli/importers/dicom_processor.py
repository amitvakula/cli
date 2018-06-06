from ..dcm import DicomFile

class DicomProcessor(object):
    def __init__(self, de_identify=False, **kwargs):
        """Process and validate each dicom in a packfile, possibly de-identifying before saving.
        
        Arguments:
            de_identify (bool): Whether or not to de-identify dicoms
        """
        self.de_identify = de_identify
        self.series_uid = None
        self.session_uid = None
        self.sop_uids = set()

    def __call__(self, path, src_file, dst_file):
        """Process an individual dicom file in a study.

        Arguments:
            path (str): The original path of the dicom file
            src_file (file): The source file
            dst_file (file): The destiniation file

        Returns:
            bool, str: True to write the file, otherwise false, and the destination path.
        """
        # Open the dicom file and parse common headers. Possibly de-identify as well.
        dcm = DicomFile(src_file, parse=True, de_identify=self.de_identify)
        if self.series_uid is not None:
            # Validate SeriesInstanceUID
            if dcm.series_uid != self.series_uid:
                log.warn('DICOM {} has a different SeriesInstanceUID ({}) from the rest of the series: {}'.format(path, dcm.series_uid, self.series_uid))
            # Validate StudyInstanceUID
            elif dcm.session_uid != self.session_uid:
                log.warn('DICOM {} has a different StudyInstanceUID ({}) from the rest of the series: {}'.format(path, dcm.session_uid, self.session_uid)) 
        else:
            self.series_uid = dcm.series_uid
            self.session_uid = dcm.session_uid
        
        # Validate SOPInstanceUID
        sop_uid = dcm.get('SOPInstanceUID')
        if sop_uid:
            if sop_uid in self.sop_uids:
                log.error('DICOM {} re-uses SOPInstanceUID {}, and will be excluded!'.format(path, sop_uid))
                return False, None
            self.sop_uids.add(sop_uid)

            # By default, name the file: {Modality}.{SOPInstanceUID}.dcm
            modality = dcm.get('Modality', 'NA')
            path = '{}.{}.dcm'.format(modality, sop_uid)

        # Write dicom to destination
        if self.de_identify:
            dcm.save(dst_file)
            return True, path

        return 'copy', path

