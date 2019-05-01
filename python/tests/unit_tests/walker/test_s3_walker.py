from unittest import mock
from flywheel_cli.walker import S3Walker
import datetime
import fs
import pytest

fs_url = 's3://bucket/path/'


@pytest.fixture
def mocked_boto3():
    mocked_boto3_patch = mock.patch('flywheel_cli.walker.s3_walker.boto3')
    yield mocked_boto3_patch.start()

    mocked_boto3_patch.stop()


@pytest.fixture
def mocked_open():
    mocked_open_patch = mock.patch('flywheel_cli.walker.s3_walker.open')
    yield mocked_open_patch.start()

    mocked_open_patch.stop()


@pytest.fixture
def mocked_os():
    mocked_os_patch = mock.patch('flywheel_cli.walker.s3_walker.os')
    yield mocked_os_patch.start()

    mocked_os_patch.stop()


@pytest.fixture
def mocked_shutil():
    mocked_shutil_patch = mock.patch('flywheel_cli.walker.s3_walker.shutil')
    yield mocked_shutil_patch.start()

    mocked_shutil_patch.stop()


@pytest.fixture
def mocked_tempfile():
    mocked_tempfile_patch = mock.patch('flywheel_cli.walker.s3_walker.tempfile')
    yield mocked_tempfile_patch.start()

    mocked_tempfile_patch.stop()


@pytest.fixture
def mocked_urlparse():
    mocked_urlparse_patch = mock.patch('flywheel_cli.walker.s3_walker.urlparse')
    yield mocked_urlparse_patch.start()

    mocked_urlparse_patch.stop()


def test_close_should_request_rmtree_from_shutil_if_tmp_dir_path_exists(mocked_boto3, mocked_urlparse, mocked_shutil):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    s3_walker = S3Walker(fs_url)
    s3_walker.tmp_dir_path = '/tmp'

    s3_walker.close()

    mocked_shutil.rmtree.assert_called_with('/tmp')


def test_close_should_not_request_rmtree_from_shutil_if_tmp_dir_path_does_not_exist(mocked_boto3,
                                                                                    mocked_urlparse, mocked_shutil):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    s3_walker = S3Walker(fs_url)

    s3_walker.close()

    mocked_shutil.rmtree.assert_not_called()


def test_get_fs_url_should_return_fs_url():
    walker = S3Walker(fs_url)

    result = walker.get_fs_url()

    assert result == fs_url


def test_init_should_request_urlparse(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')

    S3Walker(fs_url)

    mocked_urlparse.assert_called_with('s3://bucket/path/')


def test_init_should_request_client_from_boto3(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')

    S3Walker(fs_url)

    mocked_boto3.client.assert_called_with('s3')


def test_init_should_set_bucket_to_urlparse_bucket(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')

    result = S3Walker(fs_url)

    assert result.bucket == 'bucket'


def test_init_should_set_client_to_boto3_client(mocked_boto3, mocked_urlparse):
    mocked_boto3.client.return_value = {}
    mocked_urlparse.return_value = (None, 'bucket', 'path/')

    result = S3Walker(fs_url)

    assert result.client == {}


def test_init_should_set_fs_url(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')

    result = S3Walker(fs_url)

    assert result.fs_url == fs_url


def test_init_should_set_root_to_empty_string_if_url_path_is_empty(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')

    result = S3Walker(fs_url)

    assert result.root == ''


def test_init_should_set_root_to_sanitized_path_value_if_url_path_is_not_empty(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')

    result = S3Walker(fs_url)

    assert result.root == 'path'


def test_init_should_set_tmp_dir_path_to_none(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')

    result = S3Walker(fs_url)

    assert result.tmp_dir_path is None


def test_listdir_should_request_list_objects_from_client_with_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    client_mock = mock.MagicMock()
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    next( walker._listdir('/path'), None)

    client_mock.list_objects.assert_called_with(Bucket='bucket', Prefix='path/', Delimiter='/')


def test_listdir_should_request_list_objects_from_client_without_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    client_mock = mock.MagicMock()
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    next( walker._listdir('/'), None)

    client_mock.list_objects.assert_called_with(Bucket='bucket', Prefix='', Delimiter='/')


def test_listdir_should_yield_directories_if_they_exist_with_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'CommonPrefixes': [
        {'Prefix': 'path/dir1/'},
        {'Prefix': 'path/dir2/'}
    ]}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    directories = []
    for directory in walker._listdir('/path'):
        directories.append(directory)

    assert len(directories) == 2
    assert directories[0].created is None
    assert directories[0].is_dir is True
    assert directories[0].is_link is False
    assert directories[0].modified is None
    assert directories[0].name == 'dir1'
    assert directories[0].size is None
    assert directories[1].created is None
    assert directories[1].is_dir is True
    assert directories[1].is_link is False
    assert directories[1].modified is None
    assert directories[1].name == 'dir2'
    assert directories[1].size is None


def test_listdir_should_yield_directories_if_they_do_not_exist_with_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'CommonPrefixes': []}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    directories = []
    for directory in walker._listdir('/path'):
        directories.append(directory)

    assert len(directories) == 0


def test_listdir_should_yield_directories_if_they_exist_without_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'CommonPrefixes': [
        {'Prefix': 'dir1/'},
        {'Prefix': 'dir2/'}
    ]}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    directories = []
    for directory in walker._listdir('/'):
        directories.append(directory)

    assert len(directories) == 2
    assert directories[0].created is None
    assert directories[0].is_dir is True
    assert directories[0].is_link is False
    assert directories[0].modified is None
    assert directories[0].name == 'dir1'
    assert directories[0].size is None
    assert directories[1].created is None
    assert directories[1].is_dir is True
    assert directories[1].is_link is False
    assert directories[1].modified is None
    assert directories[1].name == 'dir2'
    assert directories[1].size is None


def test_listdir_should_yield_directories_if_they_do_not_exist_without_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'CommonPrefixes': []}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    directories = []
    for directory in walker._listdir('/'):
        directories.append(directory)

    assert len(directories) == 0


def test_listdir_should_yield_files_if_they_exist_with_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'Contents': [
        {'LastModified': datetime.date(1970, 1, 1), 'Key': 'path/file1.txt', 'Size': 1000},
        {'LastModified': datetime.date(1970, 1, 2), 'Key': 'path/file2.txt', 'Size': 2000}
    ]}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    files = []
    for file in walker._listdir('/path'):
        files.append(file)

    assert len(files) == 2
    assert files[0].created is None
    assert files[0].is_dir is False
    assert files[0].is_link is False
    assert files[0].modified == datetime.date(1970, 1, 1)
    assert files[0].name == 'file1.txt'
    assert files[0].size == 1000
    assert files[1].created is None
    assert files[1].is_dir is False
    assert files[1].is_link is False
    assert files[1].modified == datetime.date(1970, 1, 2)
    assert files[1].name == 'file2.txt'
    assert files[1].size == 2000


def test_listdir_should_not_yield_files_if_they_do_not_exist_with_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', 'path/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'Contents': []}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    files = []
    for file in walker._listdir('/path'):
        files.append(file)

    assert len(files) == 0


def test_listdir_should_yield_files_if_they_exist_without_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'Contents': [
        {'LastModified': datetime.date(1970, 1, 1), 'Key': 'file1.txt', 'Size': 1000},
        {'LastModified': datetime.date(1970, 1, 2), 'Key': 'file2.txt', 'Size': 2000}
    ]}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    files = []
    for file in walker._listdir('/'):
        files.append(file)

    assert len(files) == 2
    assert files[0].created is None
    assert files[0].is_dir is False
    assert files[0].is_link is False
    assert files[0].modified == datetime.date(1970, 1, 1)
    assert files[0].name == 'file1.txt'
    assert files[0].size == 1000
    assert files[1].created is None
    assert files[1].is_dir is False
    assert files[1].is_link is False
    assert files[1].modified == datetime.date(1970, 1, 2)
    assert files[1].name == 'file2.txt'
    assert files[1].size == 2000


def test_listdir_should_not_yield_files_if_they_do_not_exist_without_path(mocked_boto3, mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    client_mock = mock.MagicMock()
    client_mock.list_objects.return_value = {'Contents': []}
    mocked_boto3.client.return_value = client_mock
    walker = S3Walker(fs_url)

    files = []
    for file in walker._listdir('/'):
        files.append(file)

    assert len(files) == 0


def test_open_should_request_mkdtemp_from_tempfile_if_tmp_dir_path_does_not_exist(mocked_boto3, mocked_os,
                                                                                  mocked_shutil, mocked_tempfile,
                                                                                  mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    walker = S3Walker(fs_url)

    walker.open('/')

    mocked_tempfile.mkdtemp.assert_called()


def test_open_should_not_request_mkdtemp_from_tempfile_if_tmp_dir_path_exists(mocked_boto3, mocked_open, mocked_os,
                                                                                  mocked_shutil, mocked_tempfile,
                                                                                  mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    walker = S3Walker(fs_url)
    walker.tmp_dir_path = '/tmp'

    walker.open('/')

    mocked_tempfile.mkdtemp.assert_not_called()


def test_open_should_request_makedirs_from_os_with_root_path_if_directory_does_not_exist(mocked_boto3, mocked_open,
                                                                                         mocked_os, mocked_shutil,
                                                                                         mocked_tempfile,
                                                                                         mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    walker = S3Walker(fs_url)
    walker.tmp_dir_path = '/tmp'
    mocked_os.path.isdir.return_value = False

    walker.open('/')

    mocked_os.makedirs.assert_called_with('/tmp')


def test_open_should_request_makedirs_from_os_with_path_if_directory_does_not_exist(mocked_boto3, mocked_open,
                                                                                         mocked_os, mocked_shutil,
                                                                                         mocked_tempfile,
                                                                                         mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    walker = S3Walker(fs_url)
    walker.tmp_dir_path = '/tmp'
    mocked_os.path.isdir.return_value = False

    walker.open('/dir1/dir2/file.txt')

    mocked_os.makedirs.assert_called_with('/tmp/dir1/dir2')


def test_open_should_not_request_makedirs_from_os_if_directory_does_exist(mocked_boto3, mocked_open,
                                                                                         mocked_os, mocked_shutil,
                                                                                         mocked_tempfile,
                                                                                         mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/')
    walker = S3Walker(fs_url)
    walker.tmp_dir_path = '/tmp'
    mocked_os.path.isdir.return_value = True

    walker.open('/')

    mocked_os.makedirs.assert_not_called()


def test_open_should_request_download_file_from_boto3_client(mocked_boto3, mocked_open,
                                                             mocked_os, mocked_shutil,
                                                             mocked_tempfile,
                                                             mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/path/')
    walker = S3Walker(fs_url)
    walker.tmp_dir_path = '/tmp'

    walker.open('/dir1/file.txt')

    walker.client.download_file.assert_called_with('bucket', 'path/dir1/file.txt', '/tmp/path/dir1/file.txt')


def test_open_should_request_open(mocked_boto3, mocked_open, mocked_os, mocked_shutil, mocked_tempfile,
                                  mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/path/')
    walker = S3Walker(fs_url)
    walker.tmp_dir_path = '/tmp'

    walker.open('/dir1/file.txt')

    mocked_open.assert_called_with('/tmp/path/dir1/file.txt', 'rb')


def test_open_should_return_result_from_open(mocked_boto3, mocked_open, mocked_os, mocked_shutil, mocked_tempfile,
                                  mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/path/')
    mocked_open.return_value = {}
    walker = S3Walker(fs_url)

    result = walker.open('/')

    assert result == {}


def test_open_should_throw_if_file_if_resource_not_found_exception_is_thrown(mocked_boto3, mocked_open, mocked_os,
                                                                             mocked_shutil, mocked_tempfile,
                                                                             mocked_urlparse):
    mocked_urlparse.return_value = (None, 'bucket', '/path/')
    mocked_open.side_effect = fs.errors.ResourceNotFound('not found')
    walker = S3Walker(fs_url)

    with pytest.raises(FileNotFoundError, match=r'File /dir1/file.txt not found'):
        walker.open('/dir1/file.txt')
