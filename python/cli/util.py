
METADATA_ALIASES = {
    'group': 'group._id',
    'project': 'project.label',
    'session': 'session.label',
    'subject': 'subject.label',
    'acquisition': 'acquisition.label'
}

def set_nested_attr(obj, key, value):
    parts = key.split('.')
    for part in parts[:-1]:
        obj.setdefault(part, {})
        obj = obj[part]
    obj[parts[-1]] = value

