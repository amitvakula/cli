import io
import logging

import fs
import fs.path

from .dicom_processor import DicomProcessor
from .custom_walker import CustomWalker

log = logging.getLogger(__name__)

def create_zip_packfile(dst_file, src_fs, packfile_type=None, symlinks=False, paths=None, progress_callback=None, compression=None, **kwargs):
    """Create a zipped packfile for the given packfile_type and options, that writes a ZipFile to dst_file

    Arguments:
        dst_file (file): The destination path or file object
        src_fs (fs): The source filesystem or folder
        packfile_type (str): The packfile type, or None
        symlinks (bool): Whether or not to follow symlinks (default is False)
        progress_callback (function): Function to call with byte totals
        **kwargs: Arguments to pass to packfile process functions
    """
    import zipfile
    import zlib
    process_fn = packfile_process_fn(packfile_type, **kwargs)
    if compression is None:
        compression = zipfile.ZIP_DEFLATED
    with zipfile.ZipFile(dst_file, 'w', compression=compression) as zf:
        def write_fn(path, data):
            zf.writestr(path, data)
        create_packfile(src_fs, write_fn, process=process_fn, symlinks=symlinks, paths=paths, progress_callback=progress_callback)

def create_packfile(src_fs, write_fn, process=None, symlinks=False, paths=None, progress_callback=None):
    """Create a packfile by copying files from src_fs to dst_fs, possibly validating and/or de-identifying
    
    Arguments:
        src_fs (fs): The source filesystem
        write_fn (function): Write function that takes path and bytes to write
        process (callable): The process function that takes path, src_file, dst_file and 
            returns whether or not to write the file, and the destination path.
        symlinks (bool): Whether or not to follow symlinks (default is False)
        progress_callback (function): Function to call with byte totals
    """ 
    total_bytes = 0

    if not paths:
        # Determine file paths
        walker = CustomWalker(symlinks=symlinks)
        paths = list(walker.files(src_fs))

    for path in paths:
        write_file = 'copy'
        dst_path = path
        
        if process:
            with src_fs.open(path, 'rb', buffering=1048576) as src_file, io.BytesIO() as dst_file:
                # Process and copy or write each file
                write_file, dst_path = process(path, src_file, dst_file)
                if write_file and write_file != 'copy':
                    dst_data = dst_file.getvalue()

        if write_file == 'copy':
            dst_data = src_fs.getbytes(path)

        if write_file:
            write_fn(dst_path, dst_data)
            total_bytes = total_bytes + len(dst_data)

            if callable(progress_callback):
                progress_callback(total_bytes)


def packfile_process_fn(packfile_type, **kwargs):
    """Create a processor for the given packfile type.

    Arguments:
        packfile_type (str): The packfile type, or non
        **kwargs: The additional arguments (such as de_identify) to pass to the processor

    Returns:
        callable: The processor function, or None
    """
    if packfile_type == 'dicom':
        return DicomProcessor(**kwargs)
    return None

