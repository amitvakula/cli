
METADATA_ALIASES = {
    'group': 'group._id',
    'project': 'project.label',
    'session': 'session.label',
    'subject': 'subject.label',
    'acquisition': 'acquisition.label'
}

NO_FILE_CONTAINERS = [ 'group', 'subject' ]

def set_nested_attr(obj, key, value):
    parts = key.split('.')
    for part in parts[:-1]:
        obj.setdefault(part, {})
        obj = obj[part]
    obj[parts[-1]] = value

def sorted_container_nodes(containers):
    return sorted(containers, key=lambda x: (x.label or x.id).lower(), reverse=True)
