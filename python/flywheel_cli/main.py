#!/usr/bin/env python3
import os
import sys
import argparse

from . import legacy
from .commands import add_commands

OVERRIDE_COMMANDS = [
    'import',
    'export'
]

def main():
    legacy_invoke = False 
    legacy_commands = legacy.get_commands()

    # Remove any overridden commands here
    for cmd in OVERRIDE_COMMANDS:
        del legacy_commands[cmd]

    # Before doing argparse, check if we're invoking the legacy CLI
    argc = len(sys.argv)
    if argc > 1 and sys.argv[1] in legacy_commands:
        legacy_invoke = True

    elif argc > 2 and sys.argv[1].lower() == 'help' and sys.argv[2] in legacy_commands:
        # Forward to legacy CLI
        legacy_invoke = True

    if legacy_invoke:
        exit(legacy.invoke_command(sys.argv))

    # Create base parser and subparsers
    parser = argparse.ArgumentParser(description='Flywheel command-line interface')

    # Add commands from commands module
    add_commands(parser, legacy_commands)
    
    # Parse arguments
    args = parser.parse_args()
    
    # Additional configuration
    config_fn = getattr(args, 'config', None)
    if callable(config_fn):
        config_fn(args)

    func = getattr(args, 'func', None)
    if func is not None:
        # Invoke command
        rc = args.func(args)
        if rc is None:
            rc = 0
    else:
        parser.print_help()
        rc = 1

    exit(rc)


if __name__ == '__main__':
    main()
