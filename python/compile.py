#/usr/bin/env python3

import fnmatch
import json
import re
import os
import py_compile
import shutil
import subprocess
import urllib.request
import zipfile

from datetime import datetime

from pex.pex_builder import PEXBuilder
from pex.resolver_options import ResolverOptionsBuilder
from pex.resolver import CachingResolver
from pex.resolvable import Resolvable

SRC_DIR = os.path.abspath(os.path.dirname(__file__))
BUILD_DIR = os.path.join(SRC_DIR, 'build')
DIST_DIR = os.path.join(SRC_DIR, 'dist')
PEX_BUILD_CACHE_DIR = os.path.join(BUILD_DIR, 'pex-cache')
PACKAGE_CACHE_DIR = os.path.join(BUILD_DIR, 'src-packages')

PYTHON_VERSION = '3.6.6'
PYTHON_PACKAGES = [
    ('darwin-x86_64', 'https://storage.googleapis.com/flywheel-dist/dependencies/python-{0}/python-{0}_darwin-x86_64.tar.gz'.format(PYTHON_VERSION)),
    ('linux-x86_64', 'https://storage.googleapis.com/flywheel-dist/dependencies/python-{0}/python-{0}_linux-x86_64.tar.gz'.format(PYTHON_VERSION)),
    ('windows-x86_64', 'https://storage.googleapis.com/flywheel-dist/dependencies/python-{0}/python-{0}-embed-amd64.zip'.format(PYTHON_VERSION)),
]
PYVER = '.'.join(PYTHON_VERSION.split('.')[:2]) # Short version
PYXY = ''.join(PYTHON_VERSION.split('.')[:2])

def read_ignore_patterns():
    # Load ignores
    result = []
    pkg_ignore_path = os.path.join(SRC_DIR, 'package-ignore.txt')
    with open(pkg_ignore_path, 'r') as f:
        for line in f.readlines():
            line = line.strip()
            if line:
                result.append(re.compile(line))
    return result

def is_ignored_file(patterns, filename):
    for pattern in patterns:
        if pattern.search(filename):
            return True

    return False

def update_pth_file(path):
    if os.path.isfile(path):
        with open(path, 'a') as f:
            f.write('\r\nlib/python{}/site-packages\r\n'.format(PYVER))

def build_site_packages():
    """Use PEX to resolve dependencies in a virtual environment,
    with some customizations to reduce the size of our build.

    https://www.pantsbuild.org/pex.html
    """
    # Remove flywheel_cli from cache
    # If you skip this step, it doesn't automatically update the python code
    if os.path.isdir(PEX_BUILD_CACHE_DIR):
        for name in os.listdir(PEX_BUILD_CACHE_DIR):
            if fnmatch.fnmatch(name, 'flywheel_cli*.whl'):
                path = os.path.join(PEX_BUILD_CACHE_DIR, name)
                print('Removing {} from cache...'.format(name))
                os.remove(path)

    # Read ignore list
    # See package-ignore.txt, largely we're removing test files and
    # Multi-megabyte dicoms from the dicom folder
    ignore_patterns = read_ignore_patterns()

    # Create resolver
    # Loosely based on: https://github.com/pantsbuild/pex/blob/982cb9a988949ffff3348b9bca98ae72a0bf8847/pex/bin/pex.py#L577
    resolver_option_builder = ResolverOptionsBuilder()
    resolvables = [Resolvable.get(SRC_DIR, resolver_option_builder)]
    resolver = CachingResolver(PEX_BUILD_CACHE_DIR, None)

    # Effectively we resolve (possibly from cache) The source and all of the dependency packages
    # Then create the virtual environment, which contains those files
    print('Resolving distributions')
    resolved = resolver.resolve(resolvables)

    print('Building package lists')
    builder = PEXBuilder()
    for dist in resolved:
        builder.add_distribution(dist)
        builder.add_requirement(dist.as_requirement())

    # After this point, builder.chroot contains a full list of the files
    print('Compiling package')
    builder.freeze(bytecode_compile=False)

    site_packages_path = os.path.join(BUILD_DIR, 'site-packages.zip')

    # Create an uncompressed site-packages.zip and add all of the discovered files
    # (Except those that are filtered out)
    with open(site_packages_path, 'wb') as f:
        added_files = set()
        with zipfile.ZipFile(f, 'w') as zf:
            for filename in sorted(builder.chroot().files()):
                if is_ignored_file(ignore_patterns, filename):
                    continue

                if not filename.startswith('.deps'):
                    continue

                # Determine new path
                src_path = os.path.join(builder.chroot().chroot, filename)
                dst_path = '/'.join(filename.split('/')[2:])

                # Optionally, compile the file
                _, ext = os.path.splitext(src_path)
                if ext == '.py':
                    cfile_path = src_path + 'c'
                    dst_path += 'c'

                    print('Compiling: {}'.format(dst_path))
                    py_compile.compile(src_path, cfile=cfile_path, dfile=dst_path, optimize=1)
                    src_path = cfile_path

                if not dst_path in added_files:
                    zf.write(src_path, dst_path)
                    added_files.add(dst_path)

    return site_packages_path


if __name__ == '__main__':
    site_packages_path = build_site_packages()

    if os.path.isdir(DIST_DIR):
        shutil.rmtree(DIST_DIR)

    # Download and extract a python interpreter for each platform under build/{dist_name}
    for dist_name, package_url in PYTHON_PACKAGES:
        print('Building {}'.format(dist_name))

        # Determine the name of the package file
        package_name = os.path.basename(package_url).replace('.tar.gz', '.tgz')
        package_path = os.path.join(PACKAGE_CACHE_DIR, package_name)

        # Only perform this step if the interpreter folder doesn't already exist
        python_dist_dir = os.path.join(BUILD_DIR, dist_name)
        if not os.path.isdir(python_dist_dir):
            # Download the python interpreter if not cached
            if not os.path.isfile(package_path):
                if not os.path.isdir(PACKAGE_CACHE_DIR):
                    os.makedirs(PACKAGE_CACHE_DIR)

                with urllib.request.urlopen(package_url) as response, open(package_path, 'wb') as out:
                    shutil.copyfileobj(response, out)

            # Extract to dist folder
            os.makedirs(python_dist_dir)

            _, ext = os.path.splitext(package_name)
            if ext == '.tgz':
                subprocess.check_call(['tar', 'xf', package_path], cwd=python_dist_dir)
            else:
                subprocess.check_call(['unzip', package_path], cwd=python_dist_dir)

            # Update windows python{}._pth file:
            # See: https://docs.python.org/3/using/windows.html#finding-modules
            # Effectively we need to re-add the site-packages folder to the overridden path file
            pth_file = os.path.join(python_dist_dir, 'python{}._pth'.format(PYXY))
            update_pth_file(pth_file)

        # Zip results (with no compression, since we'll use upx on the binary)
        dist_dir = os.path.join(DIST_DIR, dist_name)
        os.makedirs(dist_dir)

        # Re-zip everything to an uncompressed zipfile
        dst_archive_path = os.path.join(dist_dir, 'python-{}.zip'.format(PYTHON_VERSION))
        with open(dst_archive_path, 'wb') as f:
            with zipfile.ZipFile(f, 'w') as zf:
                # Copy dist-dir contents
                for root, _, files in os.walk(python_dist_dir):
                    for filename in files:
                        src_path = os.path.join(root, filename)
                        arc_path = os.path.relpath(src_path, python_dist_dir)
                        zf.write(src_path, arc_path)

        # Copy site-packages
        shutil.copyfile(site_packages_path, os.path.join(dist_dir, 'site-packages.zip'))

        # Write build info, which will be used in go to determine whether or not to extract
        # the interpreter and/or the site-packages.zip
        ver_info_path = os.path.join(dist_dir, 'version.json')
        with open(ver_info_path, 'w') as f:
            json.dump({
                'python_version': PYTHON_VERSION,
                'py_ver': PYVER,
                'build_time': datetime.utcnow().isoformat()
            }, f)

