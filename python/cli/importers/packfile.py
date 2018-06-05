import fs
import io

def create_zip_packfile(dst_file, src_fs, packfile_type=None, **kwargs):
    import fs.zipfs
    """Create a Packfile instance for the given packfile_type and options, that writes a ZipFile to dst_file"""
    with fs.zipfs.ZipFS(dst_file, write=True) as dst_fs:
        create_packfile(src_fs, dst_fs)

def create_packfile(src_fs, dst_fs, de_identify=None):
    if de_identify:
        for path in src_fs.walk.files():
            with src_fs.open(path, 'rb') as src_file, io.BytesIO as dst_file:
                dst_path = self.de_identify(src_file, dst_file) or path
                dst_fs.setbytes(dst_path, dst_file.getvalue())
    else:
        # fs copy
        fs.copy.copy_fs(src_fs, dst_fs)

    src_fs.close()
    dst_fs.close()

