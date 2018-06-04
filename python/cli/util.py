import re
import os

METADATA_ALIASES = {
    'group': 'group._id',
    'project': 'project.label',
    'session': 'session.label',
    'subject': 'subject.label',
    'acquisition': 'acquisition.label'
}

NO_FILE_CONTAINERS = [ 'group', 'subject' ]

def set_nested_attr(obj, key, value):
    """Set a nested attribute in dictionary, creating sub dictionaries as necessary.

    Arguments:
        obj (dict): The top-level dictionary
        key (str): The dot-separated key
        value: The value to set
    """
    parts = key.split('.')
    for part in parts[:-1]:
        obj.setdefault(part, {})
        obj = obj[part]
    obj[parts[-1]] = value

def sorted_container_nodes(containers):
    """Returns a sorted iterable of containers sorted by label or id (whatever is available)

    Arguments:
        containers (iterable): The the list of containers to sort

    Returns:
        iterable: The sorted set of containers
    """
    return sorted(containers, key=lambda x: (x.label or x.id).lower(), reverse=True)

def to_fs_url(path):
    """Convert path to an fs url (such as osfs://~/data)

    Arguments:
        path (str): The path to convert

    Returns:
        str: A filesystem url
    """
    if path.find(':') > 1:
        # Likely a filesystem URL
        return path

    if not os.path.isdir(path):
        # Specialized path options for tar/zip files
        if re.match('^.*(\.tar|\.tgz|\.tar\.gz|\.tar\.bz2)$', path, re.I):
            return 'tar://{}'.format(path)

        _, ext = os.path.splitext(path.lower())
        if ext == '.zip':
            return 'zip://{}'.format(path)
        
    # Default is OSFS pointing at directory
    return 'osfs://{}'.format(path)

def confirmation_prompt(message):
    """Continue prompting at the terminal for a yes/no repsonse
    
    Arguments:
        message (str): The prompt message

    Returns:
        bool: True if the user responded yes, otherwise False
    """
    responses = { 'yes': True, 'y': True, 'no': False, 'n': False }
    while True:
        print('{} (yes/no): '.format(message), end='')
        choice = input().lower()
        if choice in responses:
            return responses[choice]
        print('Please respond with "yes" or "no".')


