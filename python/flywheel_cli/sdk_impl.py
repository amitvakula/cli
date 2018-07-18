"""Provides flywheel-sdk implementations of common abstract classes"""
import copy
import flywheel
import json
import os
import requests
import sys

from .importers import Uploader, ContainerResolver

CONFIG_PATH = '~/.config/flywheel/user.json'
config = None

TICKETED_UPLOAD_PATH = '/{ContainerType}/{ContainerId}/files'

def pluralize(container_type):
    """ Convert container_type to plural name
    
    Simplistic logic that supports:
    group,  project,  session, subject, acquisition, analysis, collection
    """
    if container_type == 'analysis':
        return 'analyses'
    if not container_type.endswith('s'):
        return container_type + 's'
    return container_type

def load_config():
    global config
    if config is None:
        path = os.path.expanduser(CONFIG_PATH)
        try:
            with open(path, 'r') as f:
                config = json.load(f)
        except:
            pass
    return config

def create_flywheel_client(require=True):
    config = load_config()
    if config is None or config.get('key') is None:
        if require:
            print('Not logged in, please login using `fw login` and your API key', file=sys.stderr)
            sys.exit(1)
        return None
    return flywheel.Flywheel(config['key'])

"""
For now we skip subjects, replacing them (effectively) with the project layer,
and treating them as if they always exist.
"""
class SdkUploadWrapper(Uploader, ContainerResolver):
    # TODO: Formalize subjects, please
    subject_level = 2

    def __init__(self, fw):
        self.fw = fw
        self._supports_signed_url = None
        # Session for signed-url uploads
        self._upload_session = requests.Session()

    def supports_signed_url(self):
        if self._supports_signed_url is None:
            config = self.fw.get_config()
            self._supports_signed_url = config.get('signed_url', False)
        return self._supports_signed_url

    def resolve_path(self, container_type, path):
        parts = path.split('/')

        # Remove subject level
        subject = None
        if len(parts) > self.subject_level:
            subject = parts[self.subject_level]
            del parts[self.subject_level]

        # Determine if subject exists
        if subject and container_type == 'subject':
            # Resolve project
            result = self.fw.resolve(parts)
            for child in result.children:
                if 'subject' in child and child.subject.code == subject:
                    # Return the project id
                    return result.path[-1].id, None

            return None, None

        try:
            result = self.fw.resolve(parts)
            container = result.path[-1]
            return container.id, container.get('uid')
        except flywheel.ApiException:
            return None, None

    def create_container(self, parent, container):
        if container.container_type == 'subject':
            return parent.id    

        # Create container
        create_fn = getattr(self.fw, 'add_{}'.format(container.container_type), None)
        if not create_fn:
            raise ValueError('Unsupported container type: {}'.format(container.container_type))
        create_doc = copy.deepcopy(container.context[container.container_type])

        if container.container_type == 'session':
            # Add subject to session
            create_doc['project'] = parent.id
            create_doc['subject'] = copy.deepcopy(container.context['subject'])
            # Transpose subject label to code
            create_doc['subject'].setdefault('code', create_doc['subject'].pop('label', None))
        elif parent:
            create_doc[parent.container_type] = parent.id

        new_id = create_fn(create_doc)
        return new_id
    
    def upload(self, container, name, fileobj):
        upload_fn = getattr(self.fw, 'upload_file_to_{}'.format(container.container_type), None)

        if not upload_fn:
            print('Skipping unsupported upload to container: {}'.format(container.container_type))
            return

        if self.supports_signed_url():
            self.signed_url_upload(container, name, fileobj)
        else:
            upload_fn(container.id, flywheel.FileSpec(name, fileobj))

    def signed_url_upload(self, container, name, fileobj):
        """Upload fileobj to container as name, using signed-urls"""
        # Create ticketed upload
        path_params = {
            'ContainerType': pluralize(container.container_type),
            'ContainerId': container.id
        }
        ticket, upload_url = self.create_upload_ticket(path_params, name)

        # Perform the upload
        resp = self._upload_session.put(upload_url, data=fileobj)
        resp.raise_for_status()
        resp.close()

        # Complete the upload
        self.complete_upload_ticket(path_params, ticket)

    def create_upload_ticket(self, path_params, name):
        body = {
            'metadata': {},
            'filenames': [ name ]
        }

        response = self.call_api(TICKETED_UPLOAD_PATH, 'POST',
            path_params=path_params, 
            query_params=[('ticket', '')],
            body=body,
            response_type=object
        )

        return response['ticket'], response['urls'][name]

    def complete_upload_ticket(self, path_params, ticket):
        self.call_api(TICKETED_UPLOAD_PATH, 'POST',
            path_params=path_params,
            query_params=[('ticket', ticket)])

    def call_api(self, resource_path, method, **kwargs):
        kwargs.setdefault('auth_settings', ['ApiKey'])
        kwargs.setdefault('_return_http_data_only', True)
        kwargs.setdefault('_preload_content', True)

        return self.fw.api_client.call_api(resource_path, method, **kwargs)

