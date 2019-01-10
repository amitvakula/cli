import copy
import json

from enum import Enum

class IngestState(Enum):
    """Represents the possible values for IngestState"""
    ready = 'ready'
    running = 'running'
    complete = 'complete'
    failed = 'failed'

    @staticmethod
    def ensure_state(val):
        if isinstance(val, str):
            return IngestState(val).name
        return val

class IngestItem(object):
    """Represents an item in the ingest queue.

    Attributes:
        item_id (int): The unique ID of this item.
        ingest_id (int): The ID of the ingest operation this belongs to.
        actor_id (str): The ID of the actor that is working on this item (if any).
        path (str): The source storage path of this item.
        scan_hash (str): A calculated hash that represents the state of this path as of scan time.
        ingest_type (str): The type of ingest that needs to be performed.
        state (str): The current ingest state.
        stage (str): The current processing stage.
        context (dict): The free-form context for this ingest item
    """
    def __init__(self, item_id=None, ingest_id=None, actor_id=None, path=None, ingest_type=None,
            state=None, stage=None, scan_hash=None, context=None):
        """Create an IngestItem"""
        self.item_id = item_id
        self.ingest_id = ingest_id
        self.actor_id = actor_id
        self.path = path
        self.ingest_type = ingest_type
        self.state = IngestState.ensure_state(state)
        self.stage = stage
        self.scan_hash = scan_hash
        self.context = context

    @staticmethod
    def map_field(key, value):
        """Convert from object to map for serialization"""
        if key == 'context' and value is not None:
            if isinstance(value, dict):
                return json.dumps(value)
            return value
        elif key == 'state' and isinstance(value, IngestState):
            return value.value
        return value

    @staticmethod
    def from_map(kwargs):
        """Deserialize kwargs into IngestItem.

        Args:
            kwargs (dict): The constructor arguments

        Returns:
            IngestItem: The deserialized ingest item
        """
        result = IngestItem(**kwargs)
        if result.state is not None:
            result.state = IngestState(result.state)
        if result.context is not None:
            result.context = json.loads(result.context)

        return result

