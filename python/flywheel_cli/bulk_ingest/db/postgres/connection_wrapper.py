"""Wrap database connection, to normalize differences between sqlite and psycopyg2"""

def convert_params(qs):
    """Convert from qmark to %s params"""
    return qs.replace('%', '%%').replace('?', '%s')

class CursorWrapper(object):
    """
    Provides rewriting parameters from qmark (?) to format (%s)
    and a functional lastrowid.
    """
    def __init__(self, conn, impl):
        self._conn = conn
        self._impl = impl

    def execute(self, query, args=None):
        query = convert_params(query)
        if args is None:
            args = ()
        return self._impl.execute(query, args)

    def executemany(self, query, args_seq):
        query = convert_params(query)
        args_seq = [ () if args is None else args for args in args_seq ]
        return self._impl.executemany(query, args_seq)

    def callproc(self, query, args=None):
        query = convert_params(query)
        if args is None:
            args = ()
        return self._impl.callproc(query, args)

    @property
    def lastrowid(self):
        """Return last row id by selecting lastval"""
        with self._conn.cursor() as c:
            c.execute('SELECT lastval()')
            return next(c)[0]

    def __getattr__(self, attr):
        return getattr(self._impl, attr)

    def __next__(self):
        return self._impl.__next__()

class ConnectionWrapper(object):
    """Wrapper that provides query wrapping and lastrowid for psycopg"""
    def __init__(self, impl):
        self._impl = impl

    def cursor(self):
        return CursorWrapper(self._impl, self._impl.cursor())

    def __enter__(self):
        self._impl.__enter__()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        return self._impl.__exit__(exc_type, exc_val, exc_tb)

    def __getattr__(self, attr):
        return getattr(self._impl, attr)

