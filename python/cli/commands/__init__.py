from . import import_folder

def set_subparser_print_help(parser):
    def print_help(args):
        parser.print_help()
    parser.set_defaults(func=print_help)

def print_help(default_parser, parsers):
    def print_help_fn(args):
        subcommands = ' '.join(args.subcommands)
        if subcommands in parsers:
            parsers[subcommands].print_help()
        else:
            default_parser.print_help()

    return print_help_fn

def add_commands(parser, legacy_commands):
    # map commands for help function
    parsers = {}

    # Create subparsers
    subparsers = parser.add_subparsers(title='Available commands', metavar='')

    # import
    parser_import = subparsers.add_parser('import', help='Import data into Flywheel')
    parsers['import'] = parser_import

    import_subparsers = parser_import.add_subparsers(title='Available import commands', metavar='')
    set_subparser_print_help(parser_import)

    # import folder
    parsers['import folder'] = import_folder.add_command(import_subparsers)

    # cli subcommands (stubs)
    for cmd, desc in legacy_commands.items():
        subparsers.add_parser(cmd, help=desc)

    # help commands
    parser_help = subparsers.add_parser('help')
    parser_help.add_argument('subcommands', nargs='*')
    parser_help.set_defaults(func=print_help(parser, parsers))



