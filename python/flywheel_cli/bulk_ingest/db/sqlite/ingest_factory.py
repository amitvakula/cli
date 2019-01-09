from .ingest_queue import IngestQueue
from ...abstract_ingest_factory import AbstractIngestFactory

class SqliteIngestFactory(AbstractIngestFactory):
    def __init__(self, connection_string):
        self._connection_string = connection_string

    def connect(self):
        """Create a database connection object"""
        import sqlite3
        return sqlite3.connect(self._connection_string)

    def create_queue(self):
        """Create an IngestQueue object"""
        return IngestQueue(self)
