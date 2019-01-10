import pytest

from flywheel_cli.bulk_ingest import IngestItem, IngestState

MOCK_REC = IngestItem(
    ingest_id=1,
    path='/test/folder',
    ingest_type='packfile',
    state=IngestState.ready,
    stage='initial',
    scan_hash='e59ff97941044f85df5297e1c302d260'
)


def test_ingest_queue_create(db_type, ingest_factory):
    factory = ingest_factory(db_type)

    queue = factory.create_queue()
    assert queue is not None

    queue.initialize()
    queue.initialize()  # Idempotent

def test_ingest_queue_insert(db_type, ingest_factory):
    factory = ingest_factory(db_type)
    queue = factory.create_queue()

    queue.initialize()

    rowid = queue.insert(MOCK_REC)
    assert rowid is not None

    # Verify existence
    item = queue.find(rowid)
    assert item
    assert item.item_id == rowid
    assert item.state == IngestState.ready

def test_ingest_queue_take(db_type, ingest_factory):
    factory = ingest_factory(db_type)
    queue = factory.create_queue()

    # Insert 1 item into the queue
    queue.initialize()
    rowid = queue.insert(MOCK_REC)

    # Take one item from the queue
    item = queue.pop('actor_1', wait=False)
    assert item is not None
    assert item.item_id == rowid
    assert item.state == IngestState.running
    assert item.actor_id == 'actor_1'

    # Verify that it was updated in the database
    item = queue.find(rowid)
    assert item.state == IngestState.running
    assert item.actor_id == 'actor_1'

    # Verify that the next take returns nothing
    item = queue.pop('actor_1', wait=False)
    assert item is None

def test_ingest_queue_update(db_type, ingest_factory):
    factory = ingest_factory(db_type)
    queue = factory.create_queue()

    # Insert 1 item into the queue
    queue.initialize()
    rowid = queue.insert(MOCK_REC)

    with pytest.raises(Exception):
        queue.update(rowid, foo='bar')

    queue.update(rowid, stage='scan')
    item = queue.find(rowid)
    assert item.stage == 'scan'

    # Test JSON in context
    queue.update(rowid, stage='pack', context={ 'foo': 'bar' })
    item = queue.find(rowid)
    assert item.stage == 'pack'
    assert item.context == { 'foo': 'bar' }

