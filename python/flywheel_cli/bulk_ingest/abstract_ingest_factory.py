"""Abstract factory for ingestion classes"""
import copy
from abc import ABCMeta, abstractmethod

from .ingest_controller import IngestController

class AbstractIngestFactory:
    __metaclass__ = ABCMeta

    @abstractmethod
    def connect(self):
        """Create a database connection object"""
        pass

    @abstractmethod
    def create_queue(self):
        """Create an IngestQueue object"""
        pass

    @abstractmethod
    def autoinc_column(self):
        """Get the autoincrementing id type for this database"""
        return None

    def create_autoinc_table(self, table_name, primary_key, columns):
        """Create an auto-incrementing table.

        Args:
            table_name (str): The name of the table to create
            primary_key (str): The name of the primary key column
            columns (dict): The set of columns to create
        """
        columns = copy.deepcopy(columns)
        columns[primary_key] = self.autoinc_column()

        cols = ','.join(['{} {}'.format(name, spec) for name,spec in columns.items()])

        command = 'CREATE TABLE IF NOT EXISTS {}({})'.format(table_name, cols)
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command)

    def create_controller(self):
        return IngestController(self)

    # create_reporter

