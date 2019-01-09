"""Abstract factory for ingestion classes"""
from abc import ABCMeta, abstractmethod

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

    # create_controller
    # create_reporter

