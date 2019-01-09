

class IngestItem(object):

    def __init__(self, item_id=None, ingest_id=None, actor_id=None, path=None, ingest_type=None,
            state=None, stage=None, scan_hash=None, context=None):

        """Create an IngestItem"""
        self.item_id = item_id
        self.ingest_id = ingest_id
        self.actor_id = actor_id
        self.path = path
        self.ingest_type = ingest_type
        self.state = state
        self.stage = stage
        self.scan_hash = scan_hash
        self.context = context
