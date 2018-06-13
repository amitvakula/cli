"""Provides flywheel-sdk implementations of common abstract classes"""
import copy
import flywheel
import json
import os
import sys

from .importers import Uploader, ContainerResolver

CONFIG_PATH = '~/.config/flywheel/user.json'
config = None

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
                    return result.path[-1].id

            return None

        try:
            result = self.fw.resolve(parts)
            return result.path[-1].id
        except flywheel.ApiException:
            return None

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
        # For now upload without progress...
        upload_fn = getattr(self.fw, 'upload_file_to_{}'.format(container.container_type), None)

        if not upload_fn:
            print('Skipping unsupported upload to container: {}'.format(container.container_type))
            return

        upload_fn(container.id, flywheel.FileSpec(name, fileobj))
        

