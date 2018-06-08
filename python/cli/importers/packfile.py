import fs
import io
import logging

from ..dcm import DicomFile
from .dicom_processor import DicomProcessor
from .custom_walker import CustomWalker

log = logging.getLogger(__name__)

def create_zip_packfile(dst_file, src_fs, packfile_type=None, symlinks=False, **kwargs):
    """Create a zipped packfile for the given packfile_type and options, that writes a ZipFile to dst_file

    Arguments:
        dst_file (file): The destination path or file object
        src_fs (fs): The source filesystem or folder
        packfile_type (str): The packfile type, or None
        symlinks (bool): Whether or not to follow symlinks (default is False)
        **kwargs: Arguments to pass to packfile process functions
    """
    import fs.zipfs

    process_fn = packfile_process_fn(packfile_type, **kwargs)
    with fs.zipfs.ZipFS(dst_file, write=True) as dst_fs:
        create_packfile(src_fs, dst_fs, process=process_fn, symlinks=symlinks)

def create_packfile(src_fs, dst_fs, process=None, symlinks=False):
    """Create a packfile by copying files from src_fs to dst_fs, possibly validating and/or de-identifying
    
    Arguments:
        src_fs (fs): The source filesystem
        dst_fs (fs): The destination filesystem
        process (callable): The process function that takes path, src_file, dst_file and 
            returns whether or not to write the file, and the destination path.
        symlinks (bool): Whether or not to follow symlinks (default is False)
    """ 
    walker = CustomWalker(symlinks=symlinks)

    if process:
        for path in walker.files(src_fs):
            with src_fs.open(path, 'rb') as src_file, io.BytesIO() as dst_file:
                write_file, dst_path = process(path, src_file, dst_file)
                if write_file == 'copy':
                    fs.copy.copy_file(src_fs, path, dst_fs, dst_path)
                elif write_file:
                    dst_data = dst_file.getvalue()
                    dst_fs.setbytes(dst_path, dst_data)
    else:
        # fs copy
        fs.copy.copy_fs(src_fs, dst_fs, walker=walker)

    src_fs.close()
    dst_fs.close()

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

