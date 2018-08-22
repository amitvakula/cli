# coding: utf-8

"""
    Flywheel CLI
"""
import os
import sys

from setuptools import setup, find_packages
from setuptools.command.install import install

NAME = "flywheel-cli"
VERSION = "6.0.0"

# To install the library, run the following
#
# python setup.py install
#
# prerequisite: setuptools
# http://pypi.python.org/pypi/setuptools

REQUIRES = [
    "tzlocal~=1.5.1",
    "fs~=2.0.23",
    "flywheel_migration==0.2.0.dev5",
    "flywheel-bids~=0.6.0"
]

class VerifyVersionCommand(install):
    """Custom command to verify that the git tag matches our version"""
    description = 'Verify that the git tag matches our version'

    def run(self):
        tag = os.getenv('CIRCLE_TAG')
        if tag != VERSION:
            sys.exit('Git tag: {0} does not match version: {1}'.format(tag, VERSION))

setup(
    name=NAME,
    version=VERSION,
    description="Flywheel Command Line Interface",
    author_email="support@flywheel.io",
    url="",
    keywords=["Flywheel", "flywheel", "CLI"],
    install_requires=REQUIRES,
    packages=find_packages(),
    license="MIT",
    project_urls={
        'Documentation': 'https://docs.flywheel.io/display/EM/CLI+-+Installation'
    },
    entry_points = {
        'console_scripts': [
            'fw=flywheel_cli.main:main',
        ]
    },
    cmdclass = {
        'verify': VerifyVersionCommand
    }
)
