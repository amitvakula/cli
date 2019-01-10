from ...abstract_ingest_queue import AbstractIngestQueue

class IngestQueue(AbstractIngestQueue):
    get_next_sql = '''UPDATE ingest_items SET state='running', actor_id=? WHERE item_id = (
        SELECT item_id FROM ingest_items WHERE state='ready' ORDER BY item_id LIMIT 1
        FOR UPDATE SKIP LOCKED) RETURNING *;'''

    def _get(self, actor_id):
        """Get the next item from the queue"""
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(self.get_next_sql, (actor_id,))

            try:
                row = next(c)
            except StopIteration:
                # No ready items
                row = None

            return self.deserialize(row)


