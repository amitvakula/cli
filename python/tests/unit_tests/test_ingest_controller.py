import datetime

from flywheel_cli.bulk_ingest import IngestOperation

MOCK_START_DATE = datetime.datetime.now()
MOCK_REC = IngestOperation(
    status='initial',
    root_fs='osfs:///tmp/import',
    version='1.0.0-dev.1',
    api_key='dev.flywheel.io:change-me',
    config={'foo': 'bar'},
    started=MOCK_START_DATE,
)

def test_ingest_controller_create(db_type, ingest_factory):
    factory = ingest_factory(db_type)

    controller = factory.create_controller()
    assert controller is not None

    controller.initialize()
    controller.initialize()  # Idempotent

def test_ingest_controller_insert(db_type, ingest_factory):
    factory = ingest_factory(db_type)

    controller = factory.create_controller()
    assert controller is not None

    controller.initialize()

    # Insert
    rowid = controller.insert(MOCK_REC)

    # Retrieve by id
    rec = controller.find(rowid)
    assert rec is not None
    assert rec.ingest_id == rowid
    assert rec.status == 'initial'
    assert rec.root_fs == 'osfs:///tmp/import'
    assert rec.version == '1.0.0-dev.1'
    assert rec.api_key == 'dev.flywheel.io:change-me'
    assert rec.config == {'foo': 'bar'}
    assert rec.started == MOCK_START_DATE

def test_ingest_controller_update(db_type, ingest_factory):
    factory = ingest_factory(db_type)

    controller = factory.create_controller()
    assert controller is not None

    controller.initialize()

    # Insert
    rowid = controller.insert(MOCK_REC)

    # Update
    controller.update(rowid, status='pending', config={'bar':'foo'})

    # Retrieve by id
    rec = controller.find(rowid)
    assert rec is not None
    assert rec.ingest_id == rowid
    assert rec.status == 'pending'
    assert rec.root_fs == 'osfs:///tmp/import'
    assert rec.version == '1.0.0-dev.1'
    assert rec.api_key == 'dev.flywheel.io:change-me'
    assert rec.config == {'bar': 'foo'}
    assert rec.started == MOCK_START_DATE

def test_ingest_controller_get_pending(db_type, ingest_factory):
    factory = ingest_factory(db_type)

    controller = factory.create_controller()
    assert controller is not None

    controller.initialize()

    # Insert
    rowid = controller.insert(MOCK_REC)

    # Verify pending exists
    rec = controller.get_pending()
    assert rec is not None
    assert rec.ingest_id == rowid

    # Change state to aborted
    controller.update(rowid, status='aborted')
    assert controller.get_pending() is None


