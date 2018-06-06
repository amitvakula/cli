from abc import ABC, abstractmethod
from typing.re import Pattern
from ..util import set_nested_attr, METADATA_ALIASES

class ImportTemplateNode(ABC):
    @abstractmethod
    def extract_metadata(self, name, context, current_fs=None):
        """Extract metadata from a folder-level node
        
        Arguments:
            name (str): The current folder name
            context (dict): The context object to update
            current_fs (fs): The current fs object, if available
        
        Returns:
            ImportTemplateNode: The next node in the tree if match succeeded, otherwise None
        """
        return None

class TerminalNode(ImportTemplateNode):
    """Terminal node"""
    def extract_metadata(self, name, context, current_fs=None):
        return None

TERMINAL_NODE = TerminalNode()

class StringMatchNode(ImportTemplateNode):
    def __init__(self, template=None, packfile_type=None):
        """Create a new container-level node.

        Arguments:
            template (str|re): The metavar or regular expression
            packfile_type (str): The optional packfile type if this is a packfile folder
        """
        self.template = template
        self.next_node = None

    def set_next(self, next_node):
        """Set the next node"""
        self.next_node = next_node

    def extract_metadata(self, name, context, current_fs=None):
        groups = {}

        if isinstance(self.template, Pattern):
            m = self.template.match(name)
            if not m:
                return None
            groups = m.groupdict()
        else:
            groups[self.template] = name

        for key, value in groups.items():
            if value:
                if key in METADATA_ALIASES:
                    key = METADATA_ALIASES[key]

                set_nested_attr(context, key, value)

        return self.next_node

class PackfileNode(ImportTemplateNode):
    def __init__(self, name=None, packfile_type=None):
        """Create a new packfile node that matches name.

        Arguments:
            name (str): The optional name to match
            packfile_type (str): The packfile type. If not specified, name will be used
        """
        self.name = name
        self.packfile_type = packfile_type

    def extract_metadata(self, name, context, current_fs=None):
        if self.name and self.name != name:
            return None
        
        context['packfile'] = self.packfile_type or name
        return TERMINAL_NODE 

class CompositeNode(ImportTemplateNode):
    def __init__(self):
        """Returns the first node that matches out of children."""
        self.children = []

    def add_child(self, child):
        """Add a child to the composite node

        Arguments:
            child (ImportTemplateNode): The child to add
        """
        self.children.append(child)

    def extract_metadata(self, name, context, current_fs=None):
        for child in children:
            next_node = child.extract_metadata(name, context, current_fs)
            if next_node:
                return next_node
        return None

