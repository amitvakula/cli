import collections

import fs
import pytest
import yaml

from flywheel_cli.importers import compile_regex, parse_template_string, parse_template_list, StringMatchNode, CompositeNode
from flywheel_cli.util import METADATA_EXPR

from flywheel_cli.importers import FolderImporter, StringMatchNode
from flywheel_cli.config import Config
from .test_container_factory import MockContainerResolver
from .test_folder_importer import make_config
from .test_template_string import project_pattern, session_pattern

def make_importer(resolver, template='', **kwargs):
    assert template
    template_list = yaml.load(template)

    config = make_config(resolver)
    importer = FolderImporter(config=config, **kwargs)
    importer.root_node = parse_template_list(template_list, config)

    return importer

"""
archive/PROJECT_NAME/arc001/SESSION_ID/SCANS/SEQUENCE_NUMBER/DICOM|ASSOCIATED|STIM|PMU|PRESENTATION/

where PROJECT_NAME is the name of the Project/Study
SESSION_ID is the internal number which we assign to each Imaging session
SEQUENCE_NUMBER is the chronological order of the MRI Sequence
arc001 and SCANS remain constant throughout the archive.
"""
def test_parse_yaml_template():
    importer = make_importer(MockContainerResolver(), group='unsorted', template=r"""
    - pattern: archive
    - pattern: "{project}"
    - pattern: arc001
    - pattern: "{session}"
    - pattern: SCANS
    - pattern: (?P<acquisition>\d+)
    - select:
        - pattern: DICOM
          packfile_type: dicom
        - pattern: ASSOCIATED|STIM|PRESENTATION
        - pattern: .*
          ignore: true
    """, merge_subject_and_session=True)

    result = importer.root_node

    assert result
    assert result.template.pattern == 'archive'
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == project_pattern
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == 'arc001'
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == session_pattern
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == 'SCANS'
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == r'(?P<acquisition>\d+)'
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert len(result.children) == 3

    assert result.children[0].template.pattern == 'DICOM'
    assert result.children[0].packfile_type == 'dicom'

    assert result.children[1].template.pattern == 'ASSOCIATED|STIM|PRESENTATION'
    assert result.children[1].template.match('ASSOCIATED')
    assert result.children[1].template.match('STIM')
    assert result.children[1].template.match('PRESENTATION')
    assert not result.children[1].template.match('PMU')
    assert result.children[1].packfile_type == None

    assert result.children[2].template.pattern == '.*'
    assert result.children[2].packfile_type == None
    assert result.children[2].ignore

def test_yaml_template1(mock_fs):
    # Normal discovery
    mockfs, mockfs_url = mock_fs(collections.OrderedDict({
        'archive/ASST_12345/arc001/001001/SCANS/110/DICOM': [
            '001.dcm',
            '002.dcm',
            '003.dcm'
        ],
        'archive/ASST_12345/arc001/001001/SCANS/110/PRESENTATION': [
            'rec_feedback.log'
        ],
        'archive/ASST_12345/arc001/001001/SCANS/110/ASSOCIATED': [
            'dti.bval', 'dti.bvec'
        ],
        'archive/ASST_12345/arc001/001001/SCANS/110/PMU': [
            'exclude.txt'
        ],
    }))

    resolver = MockContainerResolver()
    importer = make_importer(resolver, group='unsorted', template=r"""
    - pattern: archive
    - pattern: "{project}"
    - pattern: arc001
    - pattern: "{session}"
    - pattern: SCANS
    - pattern: (?P<acquisition>\d+)
    - select:
        - pattern: DICOM
          packfile_type: dicom
        - pattern: ASSOCIATED|STIM|PRESENTATION
        - pattern: .*
          ignore: true
    """, merge_subject_and_session=True)

    assert importer.merge_subject_and_session

    importer.discover(mockfs_url)

    itr = iter(importer.container_factory.walk_containers())

    _, child = next(itr)
    assert child.container_type == 'group'
    assert child.id == 'unsorted'

    _, child = next(itr)
    assert child.container_type == 'project'
    assert child.label == 'ASST_12345'
    assert len(child.files) == 0

    _, child = next(itr)
    assert child.container_type == 'subject'
    assert child.label == '001001'
    assert len(child.files) == 0

    _, child = next(itr)
    assert child.container_type == 'session'
    assert child.label == '001001'
    assert len(child.files) == 0

    _, child = next(itr)
    assert child.container_type == 'acquisition'
    assert child.label == '110'

    assert len(child.files) == 3
    assert '/archive/ASST_12345/arc001/001001/SCANS/110/PRESENTATION/rec_feedback.log' in child.files
    assert '/archive/ASST_12345/arc001/001001/SCANS/110/ASSOCIATED/dti.bval' in child.files
    assert '/archive/ASST_12345/arc001/001001/SCANS/110/ASSOCIATED/dti.bvec' in child.files

    assert len(child.packfiles) == 1
    desc = child.packfiles[0]
    assert desc.packfile_type == 'dicom'
    assert desc.path == '/archive/ASST_12345/arc001/001001/SCANS/110/DICOM'
    assert desc.count == 3

    try:
        next(itr)
        pytest.fail('Unexpected container found')
    except StopIteration:
        pass

