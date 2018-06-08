import copy
import re

from abc import ABC, abstractmethod
from typing.re import Pattern
from ..util import (
    METADATA_ALIASES,
    python_id_to_str,
    regex_for_property, 
    set_nested_attr, 
    str_to_python_id
)

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
            template (str|Pattern): The metavar or regular expression
            packfile_type (str): The optional packfile type if this is a packfile folder
        """
        self.template = template
        self.next_node = None
        self.packfile_type = packfile_type

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
                key = python_id_to_str(key)

                if key in METADATA_ALIASES:
                    key = METADATA_ALIASES[key]

                set_nested_attr(context, key, value)

        if self.packfile_type:
            context['packfile'] = self.packfile_type
            return TERMINAL_NODE

        return self.next_node

class CompositeNode(ImportTemplateNode):
    def __init__(self, children=None):
        """Returns the first node that matches out of children."""
        if children:
            self.children = copy.copy(children)
        else:
            self.children = []

    def add_child(self, child):
        """Add a child to the composite node

        Arguments:
            child (ImportTemplateNode): The child to add
        """
        self.children.append(child)

    def extract_metadata(self, name, context, current_fs=None):
        for child in self.children:
            next_node = child.extract_metadata(name, context, current_fs)
            if next_node:
                return next_node
        return None

def parse_template_string(value):
    """Parses a template string, creating an ImportTemplateNode tree.

    Arguments:
        value (str): The template string

    Returns:
        The created ImportTemplateNode tree
    """
    root = None
    last = None
    sections = re.split(r'(?<!\\):', value)
    for section in sections:
        parts = re.split(r'(?<!\\),', section, maxsplit=1)
        if len(parts) == 1:
            match = parts[0]
            optstr = ''
        else:
            match, optstr = parts

        # Compile the match string into a regular expression
        match = compile_regex(match)

        # Parse the options
        opts = _parse_optstr(optstr)

        # Create the next node
        node = StringMatchNode(template=match, **opts)
        if root is None:
            root = last = node
        else:
            last.set_next(node)
            last = node

    return root


IS_PROPERTY_RE = re.compile(r'^[a-z]([-_a-zA-Z0-9\.]+)([a-zA-Z0-9])$')

def compile_regex(value):
    """Compile a regular expression from a template string
    
    Arguments:
        value (str): The value to compile

    Returns:
        Pattern: The compiled regular expression
    """
    regex = ''
    escape = False
    repl = ''
    in_repl = False
    for c in value:
        if escape:
            regex = regex + '\\' + c
            escape = False
        else:
            if c == '\\':
                escape = True
            elif c == '{':
                in_repl = True
            elif c == '}':
                in_repl = False
                if IS_PROPERTY_RE.match(repl):
                    # Replace value
                    regex = regex + '(?P<{}>{})'.format(repl, regex_for_property(repl))
                else:
                    regex = regex + '{' + repl + '}'
                repl = ''
            elif in_repl:
                repl = repl + c
            else:
                regex = regex + c

    # Finally, replace group ids with valid strings
    regex = re.sub(r'(?<!\\)\(\?P<([^>]+)>', _group_str_to_id, regex)
    return re.compile(regex)

def _group_str_to_id(m):
    return '(?P<{}>'.format( str_to_python_id(m.group(1)) )

def _parse_optstr(val):
    result = {}

    pairs = val.split(',')
    for pair in pairs:
        pair = pair.strip()
        if pair:
            key, _, value = pair.partition('=')
            result[key.strip()] = value.strip()

    return result


