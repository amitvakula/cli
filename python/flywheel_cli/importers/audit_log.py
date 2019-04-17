"""Provides audit logging class"""
import csv
import os

class AuditLog(object):
    def __init__(self, audit_log_path):
        self.headers = ['Source Path', 'Flywheel Path', 'Failed', 'Message']
        self.path = audit_log_path

        if self.path and not os.path.exists(self.path):
            with open(self.path, 'w') as log_file:
                csv_writer = csv.DictWriter(log_file, fieldnames=self.headers)
                csv_writer.writeheader()

    def add_log(self, src_path, container, file_name, failed=False, message=None):
        if self.path:
            resolver_path = self.get_container_resolver_path(container, file_name)
            with open(self.path, 'a') as log_file:
                csv_writer = csv.DictWriter(log_file, fieldnames=self.headers)
                csv_writer.writerow({
                    'Source Path': src_path,
                    'Flywheel Path': resolver_path,
                    'Failed': 'true' if failed else 'false',
                    'Message': message if message else '',
                })

    def get_container_resolver_path(self, container, file_name=None):
        path = []
        while container.container_type != 'root':
            if container.container_type == 'group':
                path = [container.id] + path
            else:
                path = [container.label] + path
            container = container.parent
        if file_name is not None:
            path += ['files', file_name]
        return '/'.join(path)
