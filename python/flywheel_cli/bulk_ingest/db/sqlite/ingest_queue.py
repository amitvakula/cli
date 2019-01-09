from ...abstract_ingest_queue import AbstractIngestQueue

class IngestQueue(AbstractIngestQueue):

    select_next_sql = "SELECT * FROM ingest_items WHERE state='ready' ORDER BY item_id LIMIT 1"
    update_next_sql = "UPDATE ingest_items SET state='running', actor_id=? WHERE item_id=?"

    def create_table(self):
        """Create the table"""
        columns = self.columns.copy()

        # Use ROWID for item_id
        columns['item_id'] = 'INTEGER PRIMARY KEY'
        cols = ','.join(['{} {}'.format(name, spec) for name,spec in columns.items()])

        command = 'CREATE TABLE IF NOT EXISTS ingest_items({})'.format(cols)
        with self.connect() as conn:
            c = conn.cursor()
            c.execute(command)

    def _get(self, actor_id):
        """Get the next item from the queue"""
        # Exclusive write lock
        with self.connect() as conn:
            c = conn.cursor()
            c.execute('BEGIN IMMEDIATE')
            c.execute(self.select_next_sql)
            try:
                row = next(c)
                # Update the selected item
                c.execute(self.update_next_sql, (actor_id, row[0]))
            except StopIteration:
                # No ready items
                row = None

            if row:
                # Update locally
                result = self.deserialize(row)
                result.actor_id = actor_id
                result.state = 'running'
                return result

            return None

