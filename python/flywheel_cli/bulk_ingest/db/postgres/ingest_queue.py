from ...abstract_ingest_queue import AbstractIngestQueue

class IngestQueue(AbstractIngestQueue):
    get_next_sql = '''UPDATE ingest_items SET state='running', actor_id=? WHERE item_id = (
        SELECT item_id FROM ingest_items WHERE state='ready' ORDER BY item_id LIMIT 1
        FOR UPDATE SKIP LOCKED) RETURNING *;'''

    def create_table(self):
        """Create the table"""
        columns = self.columns.copy()

        # Use ROWID for item_id
        columns['item_id'] = 'SERIAL PRIMARY KEY'
        cols = ','.join(['{} {}'.format(name, spec) for name,spec in columns.items()])

        command = 'CREATE TABLE IF NOT EXISTS ingest_items({})'.format(cols)
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command)

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


