#!/usr/bin/env python3
import argparse
import logging
import os
import sys

from .commands import add_commands
from . import monkey

def main():
    # Global exception handler for KeyboardInterrupt
    sys.excepthook = ctrlc_excepthook

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

def ctrlc_excepthook(exctype, value, traceback):
    if exctype == KeyboardInterrupt:
        print('\nUser cancelled execution (Ctrl+C)')
        logging.getLogger().setLevel(100) # Supress any further log output
        sys.exit(1)
    else:
        sys.__excepthook__(exctype, value, traceback)

if __name__ == '__main__':
    monkey.patch_fs()
    main()
