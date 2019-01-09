
def test_connection(db_type, ingest_factory):
    factory = ingest_factory(db_type)

    # Check that we can create a connection
    # And execute a simple query
    conn = factory.connect()
    assert conn is not None

    try:
        if db_type == 'sqlite':
            command = 'select sqlite_version()'
        else:
            command = 'select version()'

        c = conn.cursor()
        c.execute(command)
        row = next(c)

        assert row[0]
        assert len(row[0]) > 0

        if db_type == 'sqlite':
            assert int(row[0].split('.')[0]) >= 3
        elif db_type == 'postgres':
            assert 'postgresql' in row[0].lower()
        else:
            assert False  # Unknown database type!
    finally:
        conn.close()
