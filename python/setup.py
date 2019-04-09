# coding: utf-8

"""
    Flywheel CLI
"""
import os
import sys

from setuptools import setup, find_packages
from setuptools.command.install import install

NAME = "flywheel-cli"

# To install the library, run the following
#
# python setup.py install
#
# prerequisite: setuptools
# http://pypi.python.org/pypi/setuptools

with open('requirements.txt') as f:
    requirements = f.read().splitlines()

with open('VERSION') as f:
    version = f.read().strip()

setup(
    name=NAME,
    version=version,
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
    }
)
