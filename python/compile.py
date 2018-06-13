import re
import os
import py_compile
import shutil
import subprocess
import urllib.request
import zipfile

from pex.pex_builder import PEXBuilder
from pex.resolver_options import ResolverOptionsBuilder
from pex.resolver import CachingResolver
from pex.resolvable import Resolvable

BUILD_DIR = os.path.abspath('build')
PEX_BUILD_CACHE_DIR = os.path.join(BUILD_DIR, 'pex-cache')
PACKAGE_CACHE_DIR = os.path.join(BUILD_DIR, 'src-packages')

PYTHON_VERSION = '3.6'
PYTHON_PACKAGES = [
    ('darwin-x86_64', 'https://storage.googleapis.com/flywheel-dist/dependencies/python-3.6.6/python-3.6.6_darwin-x86_64.tar.gz'),
    ('linux-x86_64', 'https://storage.googleapis.com/flywheel-dist/dependencies/python-3.6.6/python-3.6.6_linux-x86_64.tar.gz'),
    ('windows-x86_64', 'https://storage.googleapis.com/flywheel-dist/dependencies/python-3.6.6/python-3.6.6-embed-amd64.zip'),
]

def read_ignore_patterns():
    # Load ignores
    result = []
    with open('package-ignore.txt', 'r') as f:
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

def build_site_packages():
    # Read ignore list
    ignore_patterns = read_ignore_patterns()

    # Create resolver
    resolver_option_builder = ResolverOptionsBuilder()
    resolvables = [Resolvable.get('.', resolver_option_builder)]
    resolver = CachingResolver(PEX_BUILD_CACHE_DIR, None)

    print('Resolving distributions')
    resolved = resolver.resolve(resolvables)

    print('Building package lists')
    builder = PEXBuilder()
    for dist in resolved:
        builder.add_distribution(dist)
        builder.add_requirement(dist.as_requirement())

    print('Compiling package')
    builder.freeze(bytecode_compile=True)

    site_packages_dir = os.path.join(BUILD_DIR, 'site-packages')
    if os.path.isdir(site_packages_dir):
        shutil.rmtree(site_packages_dir)

    os.makedirs(site_packages_dir)
    for filename in sorted(builder.chroot().files()):
        if is_ignored_file(ignore_patterns, filename):
            continue

        if not filename.startswith('.deps'):
            continue

        # Determine new path
        src_path = os.path.join(builder.chroot().chroot, filename)
        filename_parts = filename.split('/')[2:]
        dst_path = os.path.join(site_packages_dir, *filename_parts)

        # Ensure that destination directory exists
        dst_dir = os.path.dirname(dst_path)
        if not os.path.isdir(dst_dir):
            os.makedirs(dst_dir)

        # Copy the file
        _, ext = os.path.splitext(src_path)
        if ext == '.py':
            dst_path += 'c'
            
            print('Compiling: {}'.format(dst_path))
            py_compile.compile(src_path, cfile=dst_path, optimize=2)
        else:
            shutil.copy2(src_path, dst_path)

    return site_packages_dir


if __name__ == '__main__':
    site_packages_dir = build_site_packages()

    # Build each distribution
    for dist_name, package_url in PYTHON_PACKAGES:
        print('Building {}'.format(dist_name))

        # Download
        package_name = os.path.basename(package_url).replace('.tar.gz', '.tgz')
        package_path = os.path.join(PACKAGE_CACHE_DIR, package_name)

        dist_dir = os.path.join(BUILD_DIR, dist_name)
        if not os.path.isdir(dist_dir):
            if not os.path.isfile(package_path):
                if not os.path.isdir(PACKAGE_CACHE_DIR):
                    os.makedirs(PACKAGE_CACHE_DIR)

                with urllib.request.urlopen(package_url) as response, open(package_path, 'wb') as out:
                    shutil.copyfileobj(response, out)

            # Extract to dist folder
            os.makedirs(dist_dir)
        
            _, ext = os.path.splitext(package_name)
            if ext == '.tgz':
                subprocess.check_call(['tar', 'xf', package_path], cwd=dist_dir)
            else:
                subprocess.check_call(['unzip', package_path], cwd=dist_dir)

        # Zip results (with no compression, since we'll use upx on the binary)
        dst_archive_path = dist_dir + '.zip'
        with open(dst_archive_path, 'wb') as f:
            with zipfile.ZipFile(f, 'w') as zf:
                # Copy dist-dir contents
                for root, _, files in os.walk(dist_dir):
                    for filename in files:
                        src_path = os.path.join(root, filename)
                        arc_path = os.path.relpath(src_path, dist_dir)
                        zf.write(src_path, arc_path)

                for root, _, files in os.walk(site_packages_dir):
                    for filename in files:
                        src_path = os.path.join(root, filename)
                        rel_path = os.path.relpath(src_path, site_packages_dir)
                        arc_path = os.path.join('lib', 'python{}'.format(PYTHON_VERSION), 'site-packages', rel_path)
                        zf.write(src_path, arc_path)

