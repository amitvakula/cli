#!/usr/bin/env python3
import os
import sys
import argparse

from .commands import add_commands

def main():
    # Create base parser and subparsers
    parser = argparse.ArgumentParser(prog='fw', description='Flywheel command-line interface')

    # Add commands from commands module
    add_commands(parser)
    
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

    sys.exit(rc)


if __name__ == '__main__':
    main()
