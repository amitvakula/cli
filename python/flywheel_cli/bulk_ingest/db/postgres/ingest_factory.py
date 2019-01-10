from .ingest_queue import IngestQueue
from .connection_wrapper import ConnectionWrapper
from ...abstract_ingest_factory import AbstractIngestFactory

class PostgresIngestFactory(AbstractIngestFactory):
    def __init__(self, connection_string):
        self._connection_string = connection_string

    def connect(self):
        """Create a database connection object"""
        import psycopg2
        conn = psycopg2.connect(self._connection_string)
        return ConnectionWrapper(conn)

    def create_queue(self):
        """Create an IngestQueue object"""
        return IngestQueue(self)

    def autoinc_column(self):
        """Get the autoincrementing id type for this database"""
        return 'SERIAL PRIMARY KEY'

