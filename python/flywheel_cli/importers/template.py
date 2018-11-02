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

from .dicom_scan import DicomScanner
from .parrec_scan import ParRecScanner

SCANNER_CLASSES = {
    'dicom': DicomScanner,
    'parrec': ParRecScanner
}

class ImportTemplateNode(ABC):
    """The node type, either folder or scanner"""
    node_type = 'folder'

    def extract_metadata(self, name, context, parent_fs=None):
        """Extract metadata from a folder-level node
        
        Arguments:
            name (str): The current folder name
            context (dict): The context object to update
            parent_fs (fs): The parent fs object, if available
        
        Returns:
            ImportTemplateNode: The next node in the tree if match succeeded, otherwise None
        """
        return None

    def scan(self, src_fs, path_prefix, context, container_factory):
        """Scan directory contents, rather than walking.

        Called if this is a scanner node.

        Arguments:
            src_fs (fs): The filesystem to scan
            path_prefix (str): The path prefix
            context (dict): The current context object
            container_factory: The container factory where nodes should be added
        """
        pass

class TerminalNode(ImportTemplateNode):
    """Terminal node"""
    def extract_metadata(self, name, context, parent_fs=None):
        return None

TERMINAL_NODE = TerminalNode()

class StringMatchNode(ImportTemplateNode):
    def __init__(self, template=None, packfile_type=None, metadata_fn=None, packfile_name=None):
        """Create a new container-level node.

        Arguments:
            template (str|Pattern): The metavar or regular expression
            packfile_type (str): The optional packfile type if this is a packfile folder
            metadata_fn (function): Optional function to extract additional metadata
            packfile_name (str): The optional packfile name, if not using the default
        """
        self.template = template
        self.next_node = None
        self.packfile_type = packfile_type
        self.metadata_fn = metadata_fn
        self.packfile_name = packfile_name

    def set_next(self, next_node):
        """Set the next node"""
        self.next_node = next_node

    def extract_metadata(self, name, context, parent_fs=None):
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

        if callable(self.metadata_fn):
            self.metadata_fn(name, context, parent_fs)

        if self.packfile_type:
            context['packfile'] = self.packfile_type
            context['packfile_name'] = self.packfile_name
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

    def extract_metadata(self, name, context, parent_fs=None):
        for child in self.children:
            next_node = child.extract_metadata(name, context, parent_fs)
            if next_node:
                return next_node
        return None

class ScannerNode(ImportTemplateNode):
    node_type = 'scanner'

    def __init__(self, config, scanner_cls):
        self.scanner = scanner_cls(config)

    def set_next(self, next_node):
        """Set the next node"""
        raise ValueError('Cannot declare nodes after dicom scanner!')

    def scan(self, src_fs, path_prefix, context, container_factory):
        """Scan directory contents, rather than walking.

        Called if this is a scanner node.

        Arguments:
            src_fs (fs): The filesystem to scan
            path_prefix (str): The path prefix
            context (dict): The current context object
            container_factory: The container factory where nodes should be added
        """
        self.scanner.discover(src_fs, context, container_factory, path_prefix=path_prefix)


def parse_template_string(value, config=None):
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
        scan = opts.pop('scan', None)

        # Create the next node
        node = StringMatchNode(template=match, **opts)
        if root is None:
            root = last = node
        else:
            last.set_next(node)
            last = node

        # Add scanner node
        if scan:
            scanner_cls = SCANNER_CLASSES.get(scan)
            if not scanner_cls:
                raise ValueError('Unknown scanner class: {}'.format(scan))
            node = ScannerNode(config, scanner_cls)
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


