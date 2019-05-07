from unittest import mock

import pytest

from flywheel_cli.walker import factory, PyFsWalker, S3Walker


@pytest.fixture
def mocked_urlparse():
    mocked_urlparse_patch = mock.patch('flywheel_cli.walker.factory.urlparse')
    yield mocked_urlparse_patch.start()

    mocked_urlparse_patch.stop()


def test_create_walker_should_request_urlparse(mocked_urlparse):
    mocked_urlparse.return_value = ('s3', 'bucket', 'path')

    factory.create_walker('s3://bucket/path/')

    mocked_urlparse.assert_called_with('s3://bucket/path/')


def test_create_walker_should_create_s3_walker_instance_for_s3_scheme(mocked_urlparse):
    mocked_urlparse.return_value = ('s3', 'bucket', 'path')

    result = factory.create_walker('s3://bucket/path/')

    assert isinstance(result, S3Walker)


def test_create_walker_should_create_pyfs_walker_instance_for_os_scheme(mocked_urlparse):
    mocked_urlparse.return_value = ('osfs', '/', '/')

    result = factory.create_walker('osfs://')

    assert isinstance(result, PyFsWalker)
