"""Provide control of an IngestOperation"""
import collections
import copy

from abc import ABCMeta, abstractmethod
from .ingest_operation import IngestOperation

class IngestController:
    """An interface for creating and controlling IngestOperations"""
    __metaclass__ = ABCMeta

    # The set of columns in this table
    columns = collections.OrderedDict([
        ('ingest_id', 'INT PRIMARY KEY'),
        ('status', 'VARCHAR(24)'),
        ('root_fs', 'VARCHAR(4096)'),
        ('version', 'VARCHAR(24)'),
        ('api_key', 'VARCHAR(256)'),
        ('config', 'TEXT'),
        ('started', 'TIMESTAMP'),
        ('folders_scanned', 'BIGINT'),
        ('files_found', 'BIGINT'),
        ('packfiles_created', 'BIGINT'),
        ('files_uploaded', 'BIGINT'),
    ])

    def __init__(self, factory):
        """Create factory instance"""
        self._factory = factory

    def connect(self):
        """Get a database connection"""
        return self._factory.connect()

    def initialize(self):
        """Ensures that the table and indexes exist, for all tables"""
        # Create ingest_operations table
        self._factory.create_autoinc_table('ingest_operations', 'ingest_id', self.columns)

        # Create ingest_items table
        queue = self._factory.create_queue()
        queue.initialize()

    def insert(self, record):
        """Insert one record, with autoincrement.

        Args:
            record (IngestOperation): The item to insert

        Returns:
            int: The inserted row id

        """
        insert_keys = list(self.columns.keys())[1:]
        insert_keys_str = ','.join(insert_keys)
        placeholders = ','.join(['?'] * len(insert_keys))

        command = 'INSERT INTO ingest_operations({}) VALUES({})'.format(insert_keys_str, placeholders)

        # Map fields ahead of insert
        params = [ IngestOperation.map_field(key, getattr(record, key, None)) for key in insert_keys ]

        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command, tuple(params))
            return c.lastrowid

    def update(self, ingest_id, **kwargs):
        """Update one record by ingest_id.

        Args:
            ingest_id (int): The id of the record to update.
            **kwargs: The set of fields to update
        """
        updates = []
        params = []

        # Create the update SET clauses
        for key, value in kwargs.items():
            if key not in self.columns:
                raise Exception('Invalid key field')
            updates.append('{} = ?'.format(key))
            params.append(IngestOperation.map_field(key, value))

        # WHERE clause argument
        params.append(ingest_id)

        command = 'UPDATE ingest_operations SET {} WHERE ingest_id = ?'.format(','.join(updates))
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command, tuple(params))

    def find(self, ingest_id):
        """Find an item by item_id"""
        return self._find_one("SELECT * FROM ingest_operations WHERE ingest_id = ?", (ingest_id,))

    def get_pending(self):
        """Check if there is any pending operation, and return it

        Return:
            IngestOperation: The pending operation, or None if there are no pending.
        """
        return self._find_one("SELECT * FROM ingest_operations WHERE status NOT IN ('complete', 'aborted') LIMIT 1")

    def _find_one(self, command, *args):
        """Find one with the given query / args"""
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command, *args)
            try:
                row = next(c)
            except StopIteration:
                row = None

        return self.deserialize(row)

    @classmethod
    def deserialize(cls, row, columns=None):
        """Deserialize a row into IngestOperation"""
        if row is None:
            return None

        if columns is None:
            columns = cls.columns.keys()

        props = {}
        for idx, colname in enumerate(columns):
            props[colname] = row[idx]

        return IngestOperation.from_map(props)
