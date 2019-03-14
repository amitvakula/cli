# coding: utf-8

"""
    Flywheel CLI
"""
import os
import sys

from setuptools import setup, find_packages
from setuptools.command.install import install

NAME = "flywheel-cli"
VERSION = "7.2.0-beta.1"

# To install the library, run the following
#
# python setup.py install
#
# prerequisite: setuptools
# http://pypi.python.org/pypi/setuptools

with open('requirements.txt') as f:
    requirements = f.read().splitlines()

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
    include_package_data=True,
    install_requires=requirements,
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
