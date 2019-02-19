import io
import logging

import fs
import fs.path
import fs.copy
from fs.zipfs import ZipFS

from ..custom_walker import CustomWalker

log = logging.getLogger(__name__)

class PackfileDescriptor(object):
    def __init__(self, packfile_type, path, count, name=None):
        """Descriptor object for creating a packfile"""
        self.packfile_type = packfile_type
        self.path = path
        self.count = count
        self.name = name

def create_zip_packfile(dst_file, src_fs, packfile_type=None, symlinks=False, paths=None, progress_callback=None, compression=None, deid_profile=None):
    """Create a zipped packfile for the given packfile_type and options, that writes a ZipFile to dst_file

    Arguments:
        dst_file (file): The destination path or file object
        src_fs (fs): The source filesystem or folder
        packfile_type (str): The packfile type, or None
        symlinks (bool): Whether or not to follow symlinks (default is False)
        progress_callback (function): Function to call with byte totals
        deid_profile: The de-identification profile to use
    """
    if compression is None:
        import zipfile
        compression = zipfile.ZIP_DEFLATED

    with ZipFS(dst_file, write=True, compression=compression) as dst_fs:
        zip_member_count = create_packfile(src_fs, dst_fs, packfile_type, symlinks=symlinks, paths=paths, progress_callback=progress_callback, deid_profile=deid_profile)

    return zip_member_count

def create_packfile(src_fs, dst_fs, packfile_type, symlinks=False, paths=None, progress_callback=None, deid_profile=None):
    """Create a packfile by copying files from src_fs to dst_fs, possibly validating and/or de-identifying

    Arguments:
        src_fs (fs): The source filesystem
        write_fn (function): Write function that takes path and bytes to write
        symlinks (bool): Whether or not to follow symlinks (default is False)
        progress_callback (function): Function to call with byte totals
        deid_profile: The de-identification profile to use
    """
    progress = {'total_bytes': 0}

    # Report progress as total_bytes
    if callable(progress_callback):
        def progress_fn(dst_fs, path):
            progress['total_bytes'] += dst_fs.getsize(path)
            progress_callback(progress['total_bytes'])
    else:
        progress_fn = None

    if not paths:
        # Determine file paths
        walker = CustomWalker(symlinks=symlinks)
        paths = list(walker.files(src_fs))

    # Attempt to de-identify using deid_profile first
    handled = False
    if deid_profile:
        if deid_profile.process_packfile(packfile_type, src_fs, dst_fs, paths, callback=progress_fn):
            return  len(paths) # Handled by de-id

    # Otherwise, just copy files into place
    for path in paths:
        # Ensure folder exists
        folder = fs.path.dirname(path)
        dst_fs.makedirs(folder, recreate=True)

        fs.copy.copy_file(src_fs, path, dst_fs, path)
        if callable(progress_fn):
            progress_fn(dst_fs, path)
    return len(paths)
