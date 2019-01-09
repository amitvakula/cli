"""Database-specific ingest logic"""
from .sqlite import SqliteIngestFactory
from .postgres import PostgresIngestFactory

# Map db_type to factory
factory_map = {
    'sqlite': SqliteIngestFactory,
    'sqlite3': SqliteIngestFactory,
    'postgres': PostgresIngestFactory,
    'postgresql': PostgresIngestFactory,
}

def create_ingest_factory(dbstr):
    """Create the appropriate ingest factory for the given connection string"""
    db_type, _, connection_string = dbstr.partition(':')

    factory_class = factory_map.get(db_type)
    if not factory_class:
        raise Exception('Unknown db provider: {}'.format(db_type))

    return factory_class(connection_string)
