import collections 
import copy

from abc import ABC, abstractmethod

CONTAINERS = ['group', 'project', 'subject', 'session', 'acquisition']

def combine_path(path, child):
    if path:
        return path + '/' + child
    return child

class ContainerNode(object):
    def __init__(self, container_type, cid=None, label=None, uid=None, parent=None, exists=False):
        self.container_type = container_type
        self.id = cid
        self.uid = uid
        self.label = label
        self.children = []
        self.exists = exists
        self.parent = parent
        self.context = None
        self.files = []
        self.packfiles = []

class ContainerResolver(ABC):
    def __init__(self):
        """Interface that handles resolution and creation of containers"""
        pass

    def path_el(self, container):
        """Get the path element for container"""
        if container.container_type == 'group':
            return container.id
        if container.id:
            return '<id:{}>'.format(container.id)
        return container.label

    @abstractmethod
    def resolve_path(self, container_type, path):
        """Resolve a container id by path
        
        Arguments:
            container_type (str): The container type hint
            path (str): The resolver path

        Returns:
            str, str: The id and uid if the container exists, otherwise None
        """
        return None, None

    @abstractmethod
    def create_container(self, parent, container):
        """Create the container described by container as a child of parent.

        Arguments:
            parent (ContainerNode): The parent container (or None if creating a group)
            container (ContainerNode): The container to create

        Returns:
            str: The id of the newly created container
        """
        return None


class ContainerFactory(object):
    def __init__(self, resolver, uids=True):
        """Manages discovery and creation of containers by looking at context objects.

        Arguments:
            resolver (ContainerResolver): The container resolver strategy
        """
        self.resolver = resolver
        self.uids = uids

        # The root container
        self.root = ContainerNode('root', exists=True)

    def is_empty(self):
        """Check if any nodes have been added to the root node"""
        return not bool(self.root.children)

    def resolve(self, context):
        """Given a context with hierarchy definitions, returns a ContainerNode if resolved.

        If the node definition is ambiguous (i.e. missing an intermediate level such as project), then this
        function returns None. This is fine if there is nothing to upload at the given node.

        Arguments:
            context (dict): The context containing the node definition

        Returns:
            ContainerNode: If the container node was resolved in the tree.
        """
        # Create containers as we go
        last = None
        current = self.root
        path = ''
        for container in CONTAINERS:
            if container in context:
                # Missing node, return None
                if current is None:
                    return None

                last = current
                current = self._resolve_child(current, container, context, path)
                path = combine_path(path, self.resolver.path_el(current))
            else:
                if current:
                    last = current
                current = None

        return current or last

    def create_containers(self):
        """Invoke resolver.create_container for each container that doesn't exist"""
        for parent, child in self.walk_containers():
            if not child.exists:
                child.id = self.resolver.create_container(parent, child)
                child.exists = True

    def walk_containers(self):
        """Breadth-first walk of containers resolved by this factory

        Yields:
            ContainerNode, ContainerNode: parent, child container nodes
        """
        queue = collections.deque() 
        for child in self.root.children:
            queue.append((None, child))

        while queue:
            parent, current = queue.popleft()

            for child in current.children:
                queue.append((current, child))

            yield parent, current

    def get_groups(self):
        """Get the top-level groups in the hierarchy

        Returns:
            list: The list of group nodes
        """
        return self.root.children

    def _resolve_child(self, parent, container_type, context, path):
        """Resolve a child by searching the parent, or creating a new node
        
        Arguments:
            parent (ContainerNode): The parent node
            container_type (str): The container type
            context (dict): The context object

        Returns:
            ContainerNode: The new or existing container node
        """
        subcon = context[container_type]
        cid = subcon.get('_id')
        uid = subcon.get('uid')
        label = subcon.get('label')

        for child in parent.children:
            # Prefer resolve by id
            if cid and child.id == cid:
                return child

            if label and child.label == label and (not self.uids or child.uid == uid):
                # In case we resolved this elsewhere, update the child id
                if cid and not child.id:
                    child.id = cid

                return child

        # Create child
        child = ContainerNode(container_type, cid=cid, label=label, uid=uid, parent=parent)
        child.context = copy.deepcopy(context)

        # Check if exists
        if self.resolver and parent.exists:
            path = combine_path(path, self.resolver.path_el(child))
            cid, uid = self.resolver.resolve_path(container_type, path)
            if cid:
                child.id = cid
                child.uid = uid
                child.exists = True

        parent.children.append(child)
        return child

