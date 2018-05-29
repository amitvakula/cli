from flywheel_cli.importers import compile_regex, parse_template_string, StringMatchNode
from flywheel_cli.util import METADATA_EXPR

def test_compile_regex():
    # No special replacement
    result = compile_regex('dicom')
    assert result.pattern == 'dicom'

    # Replace group/project
    result = compile_regex('{group}-{project._id}')
    expected = '(?P<group>{})-(?P<project__2e___id>{})'.format(METADATA_EXPR['string-id'], METADATA_EXPR['default'])
    assert result.pattern == expected

    # Don't replace a normal regex
    result = compile_regex('[a-f]{3}')
    assert result.pattern == '[a-f]{3}'

    # Ignore backslashes
    result = compile_regex(r'\w+')
    assert result.pattern == '\w+'

    # Escaped groupings
    result = compile_regex(r'\{foo\}')
    assert result.pattern == r'\{foo\}'

    # Fix groups
    result = compile_regex(r'(?P<project._id>\w+)')
    assert result.pattern == '(?P<project__2e___id>\w+)'

def test_parse_template_string():
    group_pattern = '(?P<group>{})'.format(METADATA_EXPR['string-id'])
    project_pattern = '(?P<project>{})'.format(METADATA_EXPR['default'])

    result = parse_template_string('{group}')

    assert result
    assert result.template.pattern == group_pattern
    assert result.packfile_type == None
    assert result.next_node == None

    result = parse_template_string('{group}:{project}')

    assert result
    assert result.template.pattern == group_pattern
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == project_pattern
    assert result.packfile_type == None
    assert result.next_node == None

    result = parse_template_string('{group}:{project}:(?P<session>[a-zA-Z0-9]+)-(?P<subject>\d+):scans,packfile_type=pv5')

    assert result
    assert result.template.pattern == group_pattern
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == project_pattern
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == '(?P<session>[a-zA-Z0-9]+)-(?P<subject>\d+)'
    assert result.packfile_type == None

    result = result.next_node
    assert result
    assert result.template.pattern == 'scans'
    assert result.packfile_type == 'pv5' 
    assert result.next_node == None

