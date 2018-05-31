import collections
import fs
import pytest

from cli.importers import FolderImporter
from .test_container_factory import MockContainerResolver

def mock_fs(structure):
    mockfs = fs.open_fs('mem://')

    for path, files in structure.items():
        with mockfs.makedirs(path, recreate=True) as subdir:
            for name in files:
                with subdir.open(name, 'w') as f:
                    f.write('Hello World!')

    return mockfs
    
def make_importer(resolver, group=None, project=None, no_subjects=False, no_sessions=False):
    importer = FolderImporter(resolver, group=group, project=project, 
        merge_subject_and_session=(no_subjects or no_sessions))

    if not group:
        importer.add_template_node(metavar='group')

    if not project:
        importer.add_template_node(metavar='project')

    if not no_subjects:
        importer.add_template_node(metavar='subject')

    if not no_sessions:
        importer.add_template_node(metavar='session')

    importer.add_template_node(metavar='acquisition')
    return importer

def test_folder_resolver_default():
    # Normal discovery
    mockfs = mock_fs(collections.OrderedDict({
        'scitran/Anxiety Study': [ 
            'InformedConsent_MRI.pdf', 'ScreeningForm_MRI.pdf' 
        ],
        'scitran/Anxiety Study/anx_s1/ses1': [ 
            'FractionalAnisotropy_Single_Subject.csv', 'MeanDiffusivity_Single_Subject.csv' 
        ],
        'scitran/Anxiety Study/anx_s1/ses1/T1_high-res_inplane_Ret_knk': [ 
            '8403_4_1_t1.dcm.zip'
        ],
        'scitran/Anxiety Study/anx_s1/ses1/fMRI_Ret_knk/dicom': [ 
            '001.dcm',
            '002.dcm',
            '003.dcm'
        ]
    }))

    resolver = MockContainerResolver()
    importer = make_importer(resolver)

    importer.discover(mockfs)

    itr = iter(importer.container_factory.walk_containers())

    _, child = next(itr)
    assert child.container_type == 'group'
    assert child.id == 'scitran'

    _, child = next(itr)
    assert child.container_type == 'project'
    assert child.label == 'Anxiety Study'
    assert len(child.files) == 2
    assert '/scitran/Anxiety Study/InformedConsent_MRI.pdf' in child.files
    assert '/scitran/Anxiety Study/ScreeningForm_MRI.pdf' in child.files

    _, child = next(itr)
    assert child.container_type == 'subject'
    assert child.label == 'anx_s1'

    _, child = next(itr)
    assert child.container_type == 'session'
    assert child.label == 'ses1'
    assert len(child.files) == 2
    assert '/scitran/Anxiety Study/anx_s1/ses1/FractionalAnisotropy_Single_Subject.csv' in child.files
    assert '/scitran/Anxiety Study/anx_s1/ses1/MeanDiffusivity_Single_Subject.csv' in child.files

    _, child = next(itr)
    assert child.container_type == 'acquisition'
    assert child.label == 'T1_high-res_inplane_Ret_knk'
    assert len(child.files) == 1
    assert '/scitran/Anxiety Study/anx_s1/ses1/T1_high-res_inplane_Ret_knk/8403_4_1_t1.dcm.zip' in child.files

    _, child = next(itr)
    assert child.container_type == 'acquisition'
    assert child.label == 'fMRI_Ret_knk'
    assert len(child.files) == 0
    assert len(child.packfiles) == 1
    packfile_type, files = child.packfiles[0]
    assert packfile_type == 'dicom'
    assert len(files) == 3
    assert '/scitran/Anxiety Study/anx_s1/ses1/fMRI_Ret_knk/dicom/001.dcm' in files
    assert '/scitran/Anxiety Study/anx_s1/ses1/fMRI_Ret_knk/dicom/002.dcm' in files
    assert '/scitran/Anxiety Study/anx_s1/ses1/fMRI_Ret_knk/dicom/003.dcm' in files

    try:
        next(itr)
        pytest.fail('Unexpected container found')
    except StopIteration:
        pass

def test_folder_resolver_group_and_project():
    # Normal discovery
    mockfs = mock_fs(collections.OrderedDict({
        '/': [ 
            'InformedConsent_MRI.pdf'
        ],
        'anx_s1/ses1/T1_high-res_inplane_Ret_knk': [ 
            '8403_4_1_t1.dcm.zip'
        ]
    }))

    resolver = MockContainerResolver()
    importer = make_importer(resolver, group='psychology', project='Anxiety Study')

    importer.discover(mockfs)

    itr = iter(importer.container_factory.walk_containers())

    _, child = next(itr)
    assert child.container_type == 'group'
    assert child.id == 'psychology'

    _, child = next(itr)
    assert child.container_type == 'project'
    assert child.label == 'Anxiety Study'
    assert child.files == ['/InformedConsent_MRI.pdf']

    _, child = next(itr)
    assert child.container_type == 'subject'
    assert child.label == 'anx_s1'

    _, child = next(itr)
    assert child.container_type == 'session'
    assert child.label == 'ses1'

    _, child = next(itr)
    assert child.container_type == 'acquisition'
    assert child.label == 'T1_high-res_inplane_Ret_knk'
    assert child.files == ['/anx_s1/ses1/T1_high-res_inplane_Ret_knk/8403_4_1_t1.dcm.zip']

    try:
        next(itr)
        pytest.fail('Unexpected container found')
    except StopIteration:
        pass


