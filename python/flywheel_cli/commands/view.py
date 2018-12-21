import argparse
import json
import os
import sys
import textwrap

import flywheel

from ..errors import CliError
from ..sdk_impl import create_flywheel_client, create_flywheel_session


RUN_DESC = """
Run saved or ad-hoc data-views, save new views on groups/projects/users.

examples:
  # run & save subject description view on project flywheel/Anxiety Study
  fw view run \\
      --cols subject,subject.age_years,subject.info.cohort \\
      --save 'flywheel/Anxiety Study/Subject Description' \\
      'flywheel/Anxiety Study'

  # run mean diffusivity view on the same project
  fw view run \\
      --container session \\
      --analysis-label 'afq*' \\
      --filename Mean_Diffusivity.csv \\
      'flywheel/Anxiety Study'
"""

def add_run_command(subparsers):
    """fw view run"""
    parser = subparsers.add_parser('run',
        help='Execute saved and ad-hoc data-views',
        description=RUN_DESC,
        formatter_class=argparse.RawTextHelpFormatter)

    group = parser.add_mutually_exclusive_group()
    group.add_argument('--id',
        help='Run saved data-view by id')
    group.add_argument('--label', metavar='VIEW',
        help='Run saved data-view by label (group[/proj]/label or user/label)')
    group.add_argument('--json',
        help='Run adhoc data-view with custom JSON spec (file or str)')

    parser.add_argument('--cols', metavar='COL[,COL ...]',
        help='Columns or column groups to add (separated by commas)')
    parser.add_argument('--container', metavar='CONT', dest='file_container',
        help='When matching files, the container to match on')
    parser.add_argument('--filename', metavar='NAME',
        help='When matching files, the filename pattern to match')
    parser.add_argument('--analysis-label', metavar='LBL',
        help='When matching analysis files, the label match string')
    parser.add_argument('--exclude-ids', action='store_true',
        help='Do not include id columns')
    parser.add_argument('--exclude-labels', action='store_true',
        help='Do not include label columns')

    parser.add_argument('container', metavar='CONTAINER',
        help='Container path to run data-view on (group/proj/subj/sess/acq)')
    parser.add_argument('--format', default='csv', metavar='FMT', choices=('csv', 'tsv', 'json'),
        help='Output format (csv*|tsv|json, default is csv)')
    parser.add_argument('--out', metavar='FILE',
        help='Output file to write to, default is STDOUT')

    parser.add_argument('--save', metavar='VIEW',
        help='Save ad-hoc data-view (group[/proj]/label or user/label)')

    parser.set_defaults(func=run)
    parser.set_defaults(parser=parser)
    return parser

def run(args):
    api = create_flywheel_session()
    if args.id:
        view = api.get('/views/' + args.id)
    elif args.label:
        view = lookup_view(args.label)
    elif args.json:
        if os.path.isfile(args.json):
            with open(args.json) as f:
                view = json.load(f)
        else:
            view = json.loads(args.json)
    else:
        view = {}
        if args.cols:
            view['columns'] = [{'src': col} for col in args.cols.split(',')]
        if bool(args.file_container) != bool(args.filename):
            raise CliError('--container and --filename are required together')
        elif args.analysis_label and not args.file_container:
            raise CliError('--container required when using --analysis-label')
        if args.file_container:
            view['fileSpec'] = {'container': args.file_container, 'filter': {'value': args.filename}}
        if args.analysis_label:
            view['fileSpec']['analysisFilter'] = {'label': {'value': args.analysis_label}}
        if args.exclude_ids:
            view['includeIds'] = False
        if args.exclude_labels:
            view['includeLabels'] = False

    params = {'containerId': lookup(args.container), 'format': args.format}
    if '_id' in view:
        print('Running saved data-view {}'.format(view['_id']), file=sys.stderr)
        data = api.get('/views/' + view['_id'] + '/data', params=params)
    else:
        print('Running ad-hoc data-view', file=sys.stderr)
        data = api.post('/views/data', json=view, params=params)
        if args.save:
            print('Saving data-view to ' + args.save, file=sys.stderr)
            parent_path, view['label'] = args.save.rsplit('/', 1)
            api.post('/containers/' + lookup_view_parent(parent_path) + '/views', json=view)

    outfile = sys.stdout if args.out is None else open(args.out, 'w')
    if args.format == 'json':
        json.dump(data, outfile)
    else:
        print(data, file=outfile)

def lookup(path):
    fw = create_flywheel_client()
    try:
        return fw.lookup(path).id
    except flywheel.ApiException as exc:
        raise CliError('lookup error: ' + exc.detail)

def lookup_view(path):
    """Return data-view id for 'data-view path' (eg. group/proj/view-label)"""
    parent_path, view_label = path.rsplit('/', 1)
    cont_id = lookup_view_parent(parent_path)
    api = create_flywheel_session()
    for view in api.get('/containers/' + cont_id + '/views'):
        # TODO add resolver and/or error on multiple results instead of passing 1st match
        if view['label'] == view_label:
            return view
    raise CliError('Cannot find data-view ' + path)

def lookup_view_parent(path):
    """Return container id for data-view parent path"""
    parts = path.split('/')
    if not 1 <= len(parts) <= 2:
        raise CliError('Invalid data-view parent ' + path)
    elif len(parts) == 1 and '@' in path:
        # user email - update when users switch to oid
        api = create_flywheel_session()
        api.get('/users/' + path)
        return path
    else:
        # group_id or group_id/project_label
        return lookup(path)


LS_DESC = """
List existing data-views saved on a group, project or user.

examples:
  fw view ls group_id
  fw view ls group_id/project_label
  fw view ls user@flywheel.io
"""

def add_ls_command(subparsers):
    """fw view ls"""
    parser = subparsers.add_parser('ls',
        help='List saved data-views',
        description=LS_DESC,
        formatter_class=argparse.RawTextHelpFormatter)
    parser.add_argument('container', metavar='CONTAINER',
        help='Parent container (group[/proj] or user)')
    parser.set_defaults(func=ls)
    parser.set_defaults(parser=parser)
    return parser

def ls(args):
    api = create_flywheel_session()
    cont_id = lookup_view_parent(args.container)
    views = api.get('/containers/' + cont_id + '/views')
    if views:
        for view in views:
            print('{_id} {label}'.format(**view))
    else:
        print('No views found for {}'.format(args.container))


def add_cols_command(subparsers):
    """fw view cols"""
    parser_help = 'List columns available in data-views'
    parser = subparsers.add_parser('cols',
        help=parser_help, description=parser_help + '.')
    parser.set_defaults(func=cols)
    parser.set_defaults(parser=parser)
    return parser

def cols(args):
    desc_indent_len = 28
    desc_indent = ' ' * desc_indent_len
    api = create_flywheel_session()
    for col in api.get('/views/columns'):
        col.setdefault('type', 'group' if 'group' in col else None)
        name = '{name} ({type}) '.format(**col).ljust(desc_indent_len)
        desc = col['description']
        if len(name) > desc_indent_len:
            print(name)
            print(textwrap.fill(desc, width=80, initial_indent=desc_indent, subsequent_indent=desc_indent))
        else:
            print(textwrap.fill(name + desc, width=80, subsequent_indent=desc_indent))
