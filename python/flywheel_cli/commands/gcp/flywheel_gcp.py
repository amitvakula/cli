import base64
import collections
import json
import time

import requests


class GCP:
    def __init__(self, token):
        self.rm = ResourceManager(token)
        self.gs = Storage(token)
        self.bq = BigQuery(token)
        self.hc = Healthcare(token)
        self.ml = AutoML(token)


class Service(requests.Session):
    def __init__(self, token):
        super().__init__()
        self.token = token

    def get_token(self):
        return self.token() if callable(self.token) else self.token

    @property
    def baseurl(self):
        raise NotImplementedError

    def request(self, method, url, **kwargs):
        self.headers['Authorization'] = 'Bearer ' + self.get_token()
        if not url.startswith('http'):
            url = self.baseurl + url
        response = super().request(method, url, **kwargs)
        if not response.ok:
            raise GCPError(response=response)
        return response

    def wait(self, url, done, sleep=10, timeout=60):
        start = time.time()
        response = self.get(url)
        while not done(response):
            if timeout and time.time() > start + timeout:
                message = 'Wait timeout: exceeded {}s for {}'.format(timeout, url)
                raise GCPError(response=response, message=message)
            time.sleep(sleep)
            response = self.get(url)
        return response.json()

    def wait_operation(self, response, timeout=None):
        url = self.baseurl + '/' + response.json()['name']
        operation = self.wait(url, lambda resp: resp.json().get('done'), timeout=timeout)
        if operation.get('error'):
            raise GCPError(message=operation['error']['message'])
        return operation


class GCPError(Exception):
    def __init__(self, response=None, message=None, status_code=None):
        if response == message == None:
            raise TypeError('response or message required')
        self.status_code = status_code or (response.status_code if response else 500)
        self.message = message or self.get_message(response)
        super().__init__(self.message)

    @staticmethod
    def get_message(response):
        try:
            return response.json()['error']['message']
        except:
            return response.content


class ResourceManager(Service):
    baseurl = 'https://cloudresourcemanager.googleapis.com/v1'

    def list_projects(self):
        url = '/projects'
        items = self.get(url).json().get('projects', [])
        return {item['projectId']: item for item in items}


class Storage(Service):
    baseurl = 'https://www.googleapis.com/storage/v1'

    def list_buckets(self, project):
        url = '/b?project={}'.format(project)
        items = self.get(url).json().get('items', [])
        return {item['name']: item for item in items}

    def create_bucket(self, project, bucket):
        url = '/b?project={}'.format(project)
        return self.post(url, json={'name': bucket}).json()

    def delete_bucket(self, bucket):
        url = '/b/{}'.format(bucket)
        return self.post(url).json()


class BigQuery(Service):
    baseurl = 'https://www.googleapis.com/bigquery/v2'

    def list_datasets(self, project):
        url = '/projects/{}/datasets'.format(project)
        items = self.get(url).json().get('datasets', [])
        return {item['datasetReference']['datasetId']: item for item in items}

    def create_dataset(self, project, dataset):
        url = '/projects/{}/datasets'.format(project)
        return self.post(url, json={'datasetReference': {'datasetId': dataset}}).json()

    def get_dataset(self, project, dataset):
        url = '/projects/{}/datasets/{}'.format(project, dataset)
        return self.get(url).json()

    def delete_dataset(self, project, dataset):
        url = '/projects/{}/datasets/{}'.format(project, dataset)
        return self.delete(url, params={'deleteContents': True})

    def list_tables(self, project, dataset):
        url = '/projects/{}/datasets/{}/tables'.format(project, dataset)
        items = self.get(url).json().get('tables', [])
        return {item['tableReference']['tableId'] for item in items}

    def create_table(self, project, dataset, table, **kwargs):
        url = '/projects/{}/datasets/{}/tables'.format(project, dataset)
        kwargs.update({'tableReference': {'tableId': table}})
        return self.post(url, json=kwargs).json()

    def get_table(self, project, dataset, table):
        url = '/projects/{}/datasets/{}/tables/{}'.format(project, dataset, table)
        return self.get(url).json()

    def delete_table(self, project, dataset, table):
        url = '/projects/{}/datasets/{}/tables/{}'.format(project, dataset, table)
        return self.delete(url)

    def import_from_storage(self, project, dataset, table, *gcs_globs):
        url = '/projects/{}/jobs'.format(project)
        response = self.post(url, json={'configuration': {'load': {
            'sourceUris': gcs_globs,
            'destinationTable': {'datasetId': dataset, 'tableId': table},
        }}})
        return self.wait_job(response)

    def wait_job(self, response, timeout=None):
        url = '/projects/{projectId}/jobs/{jobId}'.format(**response.json()['jobReference'])
        job = self.wait(url, lambda resp: resp.json()['status']['state'].lower() == 'done', timeout=timeout)
        if job['status'].get('errorResult'):
            raise GCPError(message=job['status']['errorResult'])
        return job

    def run_query(self, project, query):
        url = '/projects/{}/queries'.format(project)
        response = self.post(url, json={'query': query, 'useLegacySql': False})
        return BigQuery.parse_resultset(response.json())

    def get_query(self, project, query_id):
        url = '/projects/{}/queries/{}'.format(project, query_id)
        response = self.get(url)
        return BigQuery.parse_resultset(response.json())

    def insert_all(self, project, dataset, table, payload):
        url = '/projects/{}/datasets/{}/tables/{}/insertAll'.format(project, dataset, table)
        return self.post(url, json=payload)

    def upload_csv(self, project, dataset, table, csv):
        upload_url = self.baseurl.replace('bigquery', 'upload/bigquery')
        url = '{}/projects/{}/jobs?uploadType=multipart'.format(upload_url, project)
        config = {'configuration': {'load': {
            'autodetect': True,
            'destinationTable': {
                'projectId': project,
                'datasetId': dataset,
                'tableId': table},
            'ignoreUnknownValues': False,
            'maxBadRecords': 0,
            'sourceFormat': 'CSV',
            'writeDisposition': 'WRITE_TRUNCATE'}}}
        # late import to enable using module without feature/dependency
        import requests_toolbelt
        class MultipartRelatedEncoder(requests_toolbelt.MultipartEncoder):
            @property
            def content_type(self):
                return 'multipart/related; boundary={}'.format(self.boundary_value)

            def _iter_fields(self):
                for field in super()._iter_fields():
                    field.headers['Content-Disposition'] = None
                    if field.headers['Content-Type'] == 'text/csv':
                        field.headers['Content-Transfer-Encoding'] = 'base64'
                    yield field
        mpre = MultipartRelatedEncoder(fields=collections.OrderedDict({
            'conf': (None, json.dumps(config), 'application/json; charset=UTF-8'),
            'data': ('upload.csv', base64.b64encode(csv), 'text/csv'),
        }))
        headers = {'Content-Type': mpre.content_type}
        response = self.post(url, data=mpre, headers=headers)
        return self.wait_job(response)

    def truncate_table(self, project, dataset, table):
        url = '/projects/{}/jobs'.format(project)
        config = {'configuration': {'query': {
            'allowLargeResults': True,
            'destinationTable': {
                'projectId': project,
                'datasetId': dataset,
                'tableId': table},
            'priority': 'INTERACTIVE',
            'query': 'select * from {}.{} LIMIT 0'.format(dataset, table),
            'useLegacySql': False,
            'useQueryCache': False,
            'writeDisposition': 'WRITE_TRUNCATE'}}}
        response = self.post(url, json=config)
        return self.wait_job(response)

    @staticmethod
    def parse_resultset(resultset):
        fields = resultset['schema']['fields']
        return {'query_id': resultset['jobReference']['jobId'],
                'rows': [{field['name']: BigQuery.cast_field_value(field, col['v'])
                          for field, col in zip(fields, row['f'])}
                          for row in resultset.get('rows', [])]}

    @staticmethod
    def cast_field_value(field, value):
        if field['mode'] == 'REPEATED' and type(value) is list:
            return [BigQuery.cast_field_value(field, v) for v in value]
        if field['type'] == 'INTEGER':
            return int(value)
        if field['type'] == 'FLOAT':
            return float(value)
        return value


class Healthcare(Service):
    baseurl = 'https://healthcare.googleapis.com/v1alpha'

    def list_locations(self, project):
        url = '/projects/{}/locations'.format(project)
        items = self.get(url).json().get('locations', [])
        return {item['locationId']: item for item in items}

    def list_datasets(self, project, location):
        url = '/projects/{}/locations/{}/datasets'.format(project, location)
        items = self.get(url).json().get('datasets', [])
        return {item['name'].split('/')[-1]: item for item in items}

    def create_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets'.format(project, location)
        return self.post(url, params={'datasetId': dataset}).json()

    def get_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}'.format(project, location, dataset)
        return self.get(url).json()

    def delete_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}'.format(project, location, dataset)
        return self.delete(url)

    def list_dicomstores(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores'.format(project, location, dataset)
        items = self.get(url).json().get('dicomStores', [])
        return {item['name'].split('/')[-1]: item for item in items}

    def create_dicomstore(self, project, location, dataset, dicomstore):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores'.format(project, location, dataset)
        return self.post(url, params={'dicomStoreId': dicomstore}).json()

    def get_dicomstore(self, project, location, dataset, dicomstore):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}'.format(
            project, location, dataset, dicomstore)
        return self.get(url).json()

    def delete_dicomstore(self, project, location, dataset, dicomstore):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}'.format(
            project, location, dataset, dicomstore)
        return self.delete(url)

    def import_from_storage(self, project, location, dataset, dicomstore, gcs_glob):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}:import'.format(
            project, location, dataset, dicomstore)
        response = self.post(url, json={'inputConfig': {'gcsSource': {'contentUri': gcs_glob}}})
        return self.wait_operation(response)

    def export_to_storage(self, project, location, dataset, dicomstore, gcs_prefix, jpeg=False):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}:export'.format(
            project, location, dataset, dicomstore)
        gcs_destination = {'uriPrefix': gcs_prefix}
        if jpeg:
            gcs_destination['mimeType'] = 'image/jpeg; transfer-syntax=1.2.840.10008.1.2.4.50'
        response = self.post(url, json={'outputConfig': {'gcsDestination': gcs_destination}})
        return self.wait_operation(response)

    def export_to_bigquery(self, project, location, dataset, dicomstore,
                           bq_dataset=None, bq_table=None, overwrite=True):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}:export'.format(
            project, location, dataset, dicomstore)
        bq_destination = {'dataset': bq_dataset or dataset,
                          'table': bq_table or dicomstore,
                          'overwriteTable': overwrite}
        response = self.post(url, json={'outputConfig': {'bigQueryDestination': bq_destination}})
        return self.wait_operation(response)

    def get_dicomweb_client(self, project, location, dataset, dicomstore):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}/dicomWeb'.format(
            project, location, dataset, dicomstore)
        # late import to enable using module without feature/dependency
        import dicomweb_client.api
        client = dicomweb_client.api.DICOMwebClient(self.baseurl + url)
        # patch session used in dicomweb client to use tokens
        session = client._session
        def request(*args, **kwargs):
            session.headers['Authorization'] = 'Bearer ' + self.get_token()
            session.request(*args, **kwargs)
        session.request = request
        return client


class AutoML(Service):
    baseurl = 'https://automl.googleapis.com/v1beta1'

    def list_locations(self, project):
        url = '/projects/{}/locations'.format(project)
        items = self.get(url).json().get('locations', [])
        return {item['locationId']: item for item in items}

    def list_datasets(self, project, location):
        url = '/projects/{}/locations/{}/datasets'.format(project, location)
        items = self.get(url).json().get('datasets', [])
        return {item['name'].split('/')[-1]: item for item in items}

    def create_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets'.format(project, location)
        return self.post(url, json={'name': dataset}).json()

    def get_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}'.format(project, location, dataset)
        return self.get(url).json()

    def delete_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}'.format(project, location, dataset)
        return self.delete(url)

    def import_from_storage(self, project, location, dataset, inputs):
        url = '/projects/{}/locations/{}/datasets/{}:importData'.format(project, location, dataset)
        response = self.post(url, json={'gcsSource': {'inputUris': gcs_glob}})
        return self.wait_operation(response)

    def list_models(self, project, location):
        url = '/projects/{}/locations/{}/models'.format(project, location)
        items = self.get(url).json().get('models', [])
        return {item['name'].split('/')[-1]: item for item in items}

    def create_model(self, project, location, dataset, name, train_budget=1):
        url = '/projects/{}/locations/{}/models'.format(project, location)
        payload = {'datasetId': dataset, 'displayName': name,
                   'imageClassificationModelMetadata': {'trainBudget': train_budget}}
        response = self.post(url, json=payload)
        return self.wait_operation(response)

    def get_model(self, project, location, model):
        url = '/projects/{}/locations/{}/models/{}'.format(project, location, model)
        return self.get(url).json()

    def delete_model(self, project, location, model):
        url = '/projects/{}/locations/{}/models/{}'.format(project, location, model)
        return self.delete(url)

    def predict(self, project, location, model, jpeg):
        url = '/projects/{}/locations/{}/models/{}:predict'.format(project, location, model)
        payload = {'payload': {'image': {'imageBytes': base64.b64encode(jpeg)}}}
        return self.post(url, json=payload)
