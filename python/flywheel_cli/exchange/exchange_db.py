import fs
import json
import os
import requests
import sqlite3
import tempfile

from .util import GearVersionKey

DB_PATH = '~/.cache/flywheel/exchange.sqlite3'
EXCHANGE_URL='https://codeload.github.com/flywheel-io/exchange/legacy.tar.gz/master'


class GearExchangeDB(object):
    def __init__(self, path=DB_PATH):
        self.path = os.path.expanduser(path)

    def connect(self):
        path_dir = os.path.dirname(self.path)
        if not os.path.isdir(path_dir):
            os.makedirs(path_dir)

        conn = sqlite3.connect(self.path)
        c = conn.cursor()
        c.execute('CREATE TABLE IF NOT EXISTS properties(name text, value text, PRIMARY KEY(name))')
        c.execute('''CREATE TABLE IF NOT EXISTS manifests(
            filename text, folder text, name text, version text, geardoc text, PRIMARY KEY(filename))''')
        return conn

    def find_latest(self, gear_name):
        results = self.find_by_name(gear_name)
        results.sort(key=GearVersionKey, reverse=True) #Sort in descending order
        if results:
            return results[0]
        return None

    def find_version(self, gear_name, version):
        results = self.find_by_name(gear_name)
        key = GearVersionKey(version)
        for gear in self.find_by_name(gear_name):
            gear_version = GearVersionKey(gear)
            if gear_version == key:
                return gear
        return None

    def find_by_name(self, gear_name):
        result = []
        with self.connect() as conn:
            c = conn.cursor()
            c.execute('SELECT geardoc FROM manifests WHERE name=?', (gear_name,))
            while True:
                data = c.fetchone()
                if not data:
                    break
                
                result.append(json.loads(data[0]))
        return result

    def get_latest_gears(self, group=None):
        gear_map = {}
        with self.connect() as conn:
            c = conn.cursor()
            if group:
                c.execute('SELECT name, geardoc FROM manifests WHERE folder=?', (group,))
            else:
                c.execute('SELECT name, geardoc FROM manifests')

            while True:
                data = c.fetchone()
                if not data:
                    break

                gear_name = data[0]
                gear_doc = json.loads(data[1])

                if gear_name not in gear_map or GearVersionKey(gear_doc) > GearVersionKey(gear_map[gear_name]):
                    gear_map[gear_name] = gear_doc

        results = []
        for gear_name in sorted(gear_map.keys()):
            results.append(gear_map[gear_name])
        return results

    def update(self):
        conn = self.connect()

        # Use etag to determine if we need to sync
        # NOTE: (Even though github doesn't honor this for tarballs)
        c = conn.cursor()
        c.execute('SELECT value FROM properties WHERE name="etag"')

        etag = c.fetchone()
        request_headers = {}
        if etag:
            request_headers['if-none-match'] = etag[0]

        archive_path = None
        resp = requests.get(EXCHANGE_URL, headers=request_headers)
        if resp.status_code == 200:
            new_etag = resp.headers['etag']

            if etag and new_etag == etag[0]:
                # Not modified
                resp.close()
                return

            print('Updating gear list from exchange...')
            if etag:
                c.execute('UPDATE properties SET value=? where name="etag"', (new_etag,))
            else:
                c.execute('INSERT INTO properties VALUES("etag", ?)', (new_etag,))

            fd, archive_path = tempfile.mkstemp()
            with os.fdopen(fd, 'wb') as f:
                for block in resp.iter_content(16384):
                    f.write(block)

        elif resp.status_code == 304:
            # Not modified
            return
        else:
            print('Got status: {}'.format(resp.status_code))
            resp.raise_for_status()

        with fs.open_fs('tar://{}'.format(archive_path)) as exch_fs:
            # Single top level contents (e.g. flywheel-io-exchange-c04b871)
            contents_dir = list(exch_fs.listdir('/'))[0]
            with exch_fs.opendir('/{}/manifests'.format(contents_dir)) as manifest_fs:
                for entry in manifest_fs.scandir('/', namespaces=['basic']):
                    if not entry.is_dir:
                        continue

                    subdir_name = entry.name
                    with manifest_fs.opendir(subdir_name) as subdir_fs:
                        for entry in subdir_fs.scandir('/', namespaces=['basic']):
                            if entry.is_dir:
                                continue

                            filename = entry.name
                            c.execute('SELECT filename FROM manifests WHERE filename=?', (filename,))
                            if not c.fetchone():
                                # Read the manifest
                                manifest_data = subdir_fs.gettext(filename, encoding='utf-8')
                                manifest = json.loads(manifest_data)

                                c.execute('INSERT INTO manifests(filename, folder, name, version, geardoc) VALUES(?, ?, ?, ?, ?)', 
                                    (filename, subdir_name, manifest['gear']['name'], manifest['gear']['version'], manifest_data))

        conn.commit()
        conn.close()

