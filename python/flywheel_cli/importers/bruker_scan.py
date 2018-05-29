from datetime import datetime
import logging
import re

from .folder import FolderImporter
from .template import StringMatchNode
from ..bruker import extract_bruker_metadata_fn

log = logging.getLogger(__name__)

def format_timestamp_fn(dst_key):
    """Create a function to format a unix epoch string to iso8601 format.

    Arguments:
        dst_key (str): The destination key name
    
    Returns:
        function: The function that will format a datetime
    """
    def format(val):
        try:
            val = datetime.utcfromtimestamp(int(val))
            return dst_key, val.isoformat() + 'Z'
        except ValueError:
            return None
    return format

SUBJECT_PARAMS = {
    'SUBJECT_id': 'subject.label',
    'SUBJECT_study_name': 'session.label',
    'SUBJECT_abs_date': format_timestamp_fn('session.timestamp')
}

ACQP_PARAMS = {
    'ACQ_protocol_name': 'acquisition.label',
    'ACQ_abs_time': format_timestamp_fn('acquisition.timestamp')
}

def create_bruker_scanner(resolver, group, project, follow_symlinks, config, subject_pattern=None):
    """Create a bruker importer instance

    Arguments:
        resolver (ContainerResolver): The resolver instance
        group (str): The group id
        project (str): The project label
        config: (Config): The config object
        subject_pattern: (Pattern): The subject folder pattern

    Returns:
        FolderImporter: The configured folder importer instance
    """
    # Build the importer instance
    importer = FolderImporter(resolver, group=group, project=project, config=config)

    if not subject_pattern:
        subject_pattern = re.compile(r'(?P<session>[-\w]+)-\d+-(?P<subject>\d+)\..*')

    subject_meta_fn = extract_bruker_metadata_fn('subject', SUBJECT_PARAMS)
    importer.add_template_node(
        StringMatchNode(subject_pattern, metadata_fn=subject_meta_fn)
    )

    acq_metadata_fn = extract_bruker_metadata_fn('acqp', ACQP_PARAMS)
    importer.add_composite_template_node([
        StringMatchNode(re.compile('AdjResult'), packfile_type='zip', packfile_name='AdjResult.zip'),
        StringMatchNode('acquisition', packfile_type='pv5', metadata_fn=acq_metadata_fn)
    ])

    return importer

