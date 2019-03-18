import tempfile
import os
import shutil

import fs
import pytest

from flywheel_cli.bulk_ingest import db

DEFAULT_DB = ['sqlite', 'postgres']
CLEANUP_TABLES = [ 'ingest_items', 'ingest_operations' ]

def drop_tables(conn):
    with conn:
        c = conn.cursor()
        for table in CLEANUP_TABLES:
            c.execute('DROP TABLE IF EXISTS {}'.format(table))

def pytest_addoption(parser):
    parser.addoption('--db', action='append', default=[], help='list of db-modules to pass to test functions')

def pytest_generate_tests(metafunc):
    if 'db_type' in metafunc.fixturenames:
        db_opt = metafunc.config.getoption('db')
        if not db_opt:
            db_opt = DEFAULT_DB
        metafunc.parametrize('db_type', db_opt)

@pytest.fixture(scope="function")
def mock_fs():
    temp_path = tempfile.mkdtemp()
    temp_url = 'osfs://{}'.format(temp_path)
    mockfs = None

    def create_fn(structure):
        mockfs = fs.open_fs(temp_url)
        for path, files in structure.items():
            with mockfs.makedirs(path, recreate=True) as subdir:
                for name in files:
                    with subdir.open(name, 'w') as f:
                        f.write('Hello World!')

        return mockfs, temp_url

    yield create_fn

    if mockfs is not None:
        mockfs.close()

    shutil.rmtree(temp_path)

@pytest.fixture(scope="function")
def ingest_factory():
    _factories = []

    def create(db_type):
        if db_type == 'sqlite':
            connection_string = 'sqlite:/tmp/sqlite.db'
        elif db_type == 'postgres':
            connection_string = 'postgresql:' + os.environ['POSTGRES_TEST_DB']

        factory = db.create_ingest_factory(connection_string)
        _factories.append(factory)
        return factory

    yield create

    for factory in _factories:
        # Cleanup any tables that were added
        conn = factory.connect()
        try:
            drop_tables(conn)
        finally:
            conn.close()

