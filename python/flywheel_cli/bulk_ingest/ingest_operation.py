"""Provides the IngestOperation class"""
import json

class IngestOperation(object):
    """Represents a single IngestOperation.

    Attributes:
        ingest_id (int): The ID of the operation
        root_fs (str): The root filesystem path/url
        status (str): The current ingest status (scanning, paused, running, complete, aborted)
        version (str): The required software version for this operation.
        api_key (str): The API key for this ingest operation
        config (dict): The configuration for this ingest operation
        started (datetime): The time that this operation was created

        folders_scanned (int): The number of files scanned
        files_found (int): The number of uploadable files found (packfiles count as 1)
        packfiles_created (int): The number of packfiles that have been created
        files_uploaded (int): The total number of files uploaded
    """
    def __init__(self, ingest_id=None, root_fs=None, status=None, version=None, api_key=None, config=None,
            started=None, folders_scanned=0, files_found=0, packfiles_created=0, files_uploaded=0):

        """Creates a new IngestOperation"""
        self.ingest_id = ingest_id
        self.root_fs = root_fs
        self.status = status
        self.version = version
        self.api_key = api_key
        self.config = config
        self.started = started

        self.folders_scanned = folders_scanned
        self.files_found = files_found
        self.packfiles_created = packfiles_created
        self.files_uploaded = files_uploaded

    @staticmethod
    def map_field(key, value):
        """Convert from object to map for serialization"""
        if key == 'config' and value is not None:
            if isinstance(value, dict):
                return json.dumps(value)
        return value

    @staticmethod
    def from_map(kwargs):
        """Deserialize kwargs into IngestOperation.

        Args:
            kwargs (dict): The constructor arguments

        Returns:
            IngestOperation: The deserialized ingest operation
        """
        result = IngestOperation(**kwargs)
        if result.config is not None:
            result.config = json.loads(result.config)
        return result

