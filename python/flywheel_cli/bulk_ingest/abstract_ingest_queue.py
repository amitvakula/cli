import collections
import time

from .ingest_item import IngestItem

from abc import ABCMeta, abstractmethod

class AbstractIngestQueue:
    """Abstract class that provides queue behavior around ingest items"""
    __metaclass__ = ABCMeta

    # The set of columns in this table
    columns = collections.OrderedDict([
        ('item_id', 'INT'),
        ('ingest_id', 'INT'),
        ('actor_id', 'VARCHAR(128)'),
        ('path', 'VARCHAR(4096)'),
        ('ingest_type', 'VARCHAR(24)'),
        ('state', 'VARCHAR(24)'),
        ('stage', 'VARCHAR(24)'),
        ('scan_hash', 'CHAR(32)'),
        ('context', 'TEXT'),
    ])
    # The set of indexes to create for this table
    indexes = [
        ('state_idx', ['state']),
        ('actor_idx', ['actor_id']),
    ]

    def __init__(self, factory):
        """Create an AbstractIngestQueue

        Args:
            factory: The connection factory
        """
        self._factory = factory

    def connect(self):
        """Get a database connection"""
        return self._factory.connect()

    def initialize(self):
        """Ensures that the table and indexes exist"""
        self.create_table()
        self.create_indexes()

    def insert(self, record):
        """Insert one record, with autoincrement.

        Args:
            record (IngestItem): The item to insert

        Returns:
            int: The inserted row id

        """
        insert_keys = list(self.columns.keys())[1:]
        insert_keys_str = ','.join(insert_keys)
        placeholders = ','.join(['?'] * len(insert_keys))

        command = 'INSERT INTO ingest_items({}) VALUES({})'.format(insert_keys_str, placeholders)

        # Map fields ahead of insert
        params = [ IngestItem.map_field(key, getattr(record, key, None)) for key in insert_keys ]

        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command, tuple(params))
            return c.lastrowid

    def update(self, item_id, **kwargs):
        """Update one record by item_id.

        Args:
            item_id (int): The id of the record to update.
            **kwargs: The set of fields to update
        """
        updates = []
        params = []

        # Create the update SET clauses
        for key, value in kwargs.items():
            if key not in self.columns:
                raise Exception('Invalid key field')
            updates.append('{} = ?'.format(key))
            params.append(IngestItem.map_field(key, value))

        # WHERE clause argument
        params.append(item_id)

        command = 'UPDATE ingest_items SET {} WHERE item_id = ?'.format(','.join(updates))
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command, tuple(params))

    def create_indexes(self):
        """Create necessary indexes"""
        for index_name, columns in self.indexes:
            command = 'CREATE INDEX IF NOT EXISTS {} on ingest_items({})'.format(index_name, ','.join(columns))
            with self.connect() as conn:
                c = conn.cursor()
                c.execute(command)

    def pop(self, actor_id, wait=True, wait_time=1.0):
        """Get the next item off the work queue"""
        while True:
            item = self._get(actor_id)
            if item is not None or not wait:
                return item
            time.sleep(wait_time)

    def find(self, item_id):
        """Find an item by item_id"""
        command = 'SELECT * FROM ingest_items WHERE item_id = ?'
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command, (item_id,))

            try:
                row = next(c)
            except StopIteration:
                row = None

            return self.deserialize(row)

    def deserialize(self, row, columns=None):
        """Deserialize a row into IngestItem"""
        if row is None:
            return None

        if columns is None:
            columns = self.columns.keys()

        props = {}
        for idx, colname in enumerate(columns):
            props[colname] = row[idx]

        return IngestItem.from_map(props)

    @abstractmethod
    def create_table(self):
        """Create the table"""
        pass

    @abstractmethod
    def _get(self, actor_id):
        """Get the next item from the queue, returning None if the queue is empty"""
        pass
