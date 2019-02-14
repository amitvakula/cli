import base64
import collections
import json
import time

import mimetypes
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
        self.headers['Content-Type'] = 'application/json; charset=utf-8'
        if not url.startswith('http'):
            url = self.baseurl + url
        response = super().request(method, url, **kwargs)

        transient_failure_retries = 0
        while response.status_code == 429 or 500 <= response.status_code <= 599 and transient_failure_retries <= 5:
            time.sleep(2 ** transient_failure_retries)
            response = super().request(method, url, **kwargs)
            transient_failure_retries += 1

        if not response.ok:
            raise GCPError(response=response)
        return response

    def wait(self, url, done, sleep=10, timeout=None):
        start = time.time()
        response = self.get(url)
        while not done(response):
            if timeout is not None and time.time() > start + timeout:
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
        self.status_code = status_code or (response.status_code if response else None)
        self.message = message or self.get_message(response)
        if self.status_code is not None:
            self.message = '{}: {}'.format(self.status_code, self.message)
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

    def list_objects(self, bucket):
        url = '/b/{}/o'.format(bucket)
        return self.get(url).json()

    def upload_object(self, bucket, file_, name=None, mime=None):
        file_ = file_ if hasattr(file_, 'read') else open(file_, 'rb')
        if name is None:
            name = file_.name
        if mime is None:
            mime = mimetypes.guess_type(name)[0] or 'application/octet-stream'
        upload_url = self.baseurl.replace('storage', 'upload/storage')
        url = '{}/b/{}/o?uploadType=media&name={}'.format(upload_url, bucket, name)
        return self.post(url, data=file_, headers={'Content-Type': mime}).json()


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

    def import_from_storage(self, project, dataset, table, gcs_globs, timeout=None):
        url = '/projects/{}/jobs'.format(project)
        response = self.post(url, json={'configuration': {'load': {
            'sourceUris': gcs_globs if isinstance(gcs_globs, list) else [gcs_globs],
            'destinationTable': {'datasetId': dataset, 'tableId': table},
        }}})
        return self.wait_job(response, timeout=timeout)

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

    def upload_csv(self, project, dataset, table, csv, timeout=None):
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
        return self.wait_job(response, timeout=timeout)

    def truncate_table(self, project, dataset, table, timeout=None):
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
        return self.wait_job(response, timeout=timeout)

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

    def import_from_storage(self, project, location, dataset, dicomstore, gcs_glob, timeout=None):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}:import'.format(
            project, location, dataset, dicomstore)
        response = self.post(url, json={'inputConfig': {'gcsSource': {'contentUri': gcs_glob}}})
        return self.wait_operation(response, timeout=timeout)

    def export_to_storage(self, project, location, dataset, dicomstore, gcs_prefix, jpeg=False, timeout=None):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}:export'.format(
            project, location, dataset, dicomstore)
        gcs_destination = {'uriPrefix': gcs_prefix}
        if jpeg:
            gcs_destination['mimeType'] = 'image/jpeg; transfer-syntax=1.2.840.10008.1.2.4.50'
        response = self.post(url, json={'outputConfig': {'gcsDestination': gcs_destination}})
        return self.wait_operation(response, timeout=timeout)

    def export_to_bigquery(self, project, location, dataset, dicomstore,
                           bq_dataset=None, bq_table=None, overwrite=True, timeout=None):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}:export'.format(
            project, location, dataset, dicomstore)
        bq_destination = {'dataset': bq_dataset or dataset,
                          'table': bq_table or dicomstore,
                          'overwriteTable': overwrite}
        response = self.post(url, json={'outputConfig': {'bigQueryDestination': bq_destination}})
        return self.wait_operation(response, timeout=timeout)

    def get_dicomweb_client(self, project, location, dataset, dicomstore):
        url = '/projects/{}/locations/{}/datasets/{}/dicomStores/{}/dicomWeb'.format(
            project, location, dataset, dicomstore)
        # late import to enable using module without feature/dependency
        import dicomweb_client.api
        client = dicomweb_client.api.DICOMwebClient(self.baseurl + url)
        # patch session used in dicomweb client to use tokens
        session = client._session
        request = session.request
        def auth_request(*args, **kwargs):
            session.headers['Authorization'] = 'Bearer ' + self.get_token()
            return request(*args, **kwargs)
        session.request = auth_request
        return client

    def list_hl7_stores(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores'.format(
            project, location, dataset)
        return self.get(url).json()

    def create_hl7_store(self, project, location, dataset, hl7_store):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores'.format(
            project, location, dataset)
        return self.post(url, params={'hl7V2StoreId': hl7_store}).json()

    def get_hl7_store(self, project, location, dataset, hl7_store):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores/{}'.format(
            project, location, dataset, hl7_store)
        return self.get(url).json()

    def delete_hl7_store(self, project, location, dataset, hl7_store):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores/{}'.format(
            project, location, dataset, hl7_store)
        return self.delete(url).json()

    def list_hl7_messages(self, project, location, dataset, hl7_store, query=''):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores/{}/messages/{}'.format(
            project, location, dataset, hl7_store, query)
        return self.get(url).json()

    def create_hl7_message(self, project, location, dataset, hl7_store, msg):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores/{}/messages'.format(
            project, location, dataset, hl7_store)
        return self.post(url, json={'message': {'data': msg}}).json()

    def get_hl7_message(self, project, location, dataset, hl7_store, msg_id):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores/{}/messages/{}'.format(
            project, location, dataset, hl7_store, msg_id)
        return self.get(url).json()

    def delete_hl7_message(self, project, location, dataset, hl7_store, msg_id):
        url = '/projects/{}/locations/{}/datasets/{}/hl7V2Stores/{}/messages/{}'.format(
            project, location, dataset, hl7_store, msg_id)
        return self.delete(url).json()

    def list_fhir_stores(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}/fhirStores'.format(
            project, location, dataset)
        return self.get(url).json()

    def create_fhir_store(self, project, location, dataset, fhir_store):
        url = '/projects/{}/locations/{}/datasets/{}/fhirStores'.format(
            project, location, dataset)
        return self.post(url, params={'fhirStoreId': fhir_store}).json()

    def get_fhir_store(self, project, location, dataset, fhir_store):
        url = '/projects/{}/locations/{}/datasets/{}/fhirStores/{}'.format(
            project, location, dataset, fhir_store)
        return self.get(url).json()

    def delete_fhir_store(self, project, location, dataset, fhir_store):
        url = '/projects/{}/locations/{}/datasets/{}/fhirStores/{}'.format(
            project, location, dataset, fhir_store)
        return self.delete(url).json()

    def create_fhir_resource(self, project, location, dataset, fhir_store, payload):
        url = '/projects/{}/locations/{}/datasets/{}/fhirStores/{}/resources/{}'.format(
            project, location, dataset, fhir_store, payload['resourceType'])
        return self.post(url, json=payload).json()

    def list_fhir_resources(self, project, location, dataset, fhir_store, resource_type=None, query=None):
        query_part = ''
        if resource_type:
            query_part += '{}/'.format(resource_type)

        if query:
            query_part += '?{}'.format(query)

        url = '/projects/{}/locations/{}/datasets/{}/fhirStores/{}/resources/{}'.format(
            project, location, dataset, fhir_store, query_part)
        return self.get(url).json()

    def get_fhir_resource(self, project, location, dataset, fhir_store, resource_type, id):
        url = '/projects/{}/locations/{}/datasets/{}/fhirStores/{}/resources/{}/{}'.format(
            project, location, dataset, fhir_store, resource_type, id)
        return self.get(url).json()


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

    def create_dataset(self, project, location, display_name, classification_type='MULTICLASS'):
        url = '/projects/{}/locations/{}/datasets'.format(project, location)
        payload = {'displayName': display_name,
                   'imageClassificationDatasetMetadata': {'classificationType': classification_type}}
        return self.post(url, json=payload).json()

    def get_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}'.format(project, location, dataset)
        return self.get(url).json()

    def delete_dataset(self, project, location, dataset):
        url = '/projects/{}/locations/{}/datasets/{}'.format(project, location, dataset)
        return self.delete(url)

    def import_from_storage(self, project, location, dataset, inputs, timeout=None):
        url = '/projects/{}/locations/{}/datasets/{}:importData'.format(project, location, dataset)
        response = self.post(url, json={'inputConfig': {'gcsSource': {'inputUris': inputs}}})
        return self.wait_operation(response, timeout=timeout)

    def list_models(self, project, location):
        url = '/projects/{}/locations/{}/models'.format(project, location)
        items = self.get(url).json().get('model', [])
        return {item['name'].split('/')[-1]: item for item in items}

    def create_model(self, project, location, dataset, display_name, train_budget=1, timeout=None):
        url = '/projects/{}/locations/{}/models'.format(project, location)
        payload = {'datasetId': dataset, 'displayName': display_name,
                   'imageClassificationModelMetadata': {'trainBudget': train_budget}}
        response = self.post(url, json=payload)
        return self.wait_operation(response, timeout=timeout)

    def get_model(self, project, location, model):
        url = '/projects/{}/locations/{}/models/{}'.format(project, location, model)
        return self.get(url).json()

    def get_model_evaluation(self, project, location, dataset, model):
        url = '/projects/{}/locations/{}/models/{}/modelEvaluations'.format(project, location, model)
        return self.get(url).json()

    def delete_model(self, project, location, model):
        url = '/projects/{}/locations/{}/models/{}'.format(project, location, model)
        return self.delete(url)

    def predict(self, project, location, model, image, score_threshold=0.5):
        url = '/projects/{}/locations/{}/models/{}:predict'.format(project, location, model)
        # TODO figure out score_threshold mechanics:
        # https://cloud.google.com/vision/automl/docs/reference/rest/v1beta1/projects.locations.models/predict
        # Invalid value at 'params[0].value' (TYPE_STRING), 0.85
        payload = {'payload': {'image': {'imageBytes': base64.b64encode(image).decode('utf8')}}}
        return self.post(url, json=payload).json()
