import tempfile

from abc import ABC, abstractmethod

from .work_queue import Task, WorkQueue
from .packfile import create_zip_packfile
from .progress_reporter import ProgressReporter

MAX_IN_MEMORY_XFER = 32 * (2 ** 20) # Files under 32mb send as one chunk

class Uploader(ABC):
    verb = 'Uploading'

    """Abstract uploader class, that can upload files"""
    @abstractmethod
    def upload(self, container, name, fileobj):
        """Upload the given file-like object to the given container as name.

        Arguments:
            container (ContainerNode): The destination container
            name (str): The file name
            fileobj (obj): The file-like object, which supports read()

        Yields:
            int: Number of bytes uploaded (periodically)
        """
        pass

    def supports_signed_url(self):
        """Check if signed url upload is supported.

        Returns:
            bool: True if signed url upload is supported
        """
        return False

class UploadFileWrapper(object):
    """Wrapper around file that measures progress"""
    def __init__(self, fileobj):
        self.fileobj = fileobj
        self._sent = 0

        self.fileobj.seek(0,2)
        self._total_size = self.fileobj.tell()
        self.fileobj.seek(0)

    def read(self, size=-1):
        chunk = self.fileobj.read(size)
        self._sent = self._sent + len(chunk)
        return chunk

    def reset(self):
        self.fileobj.seek(0)

    @property
    def len(self):
        return (self._total_size - self._sent)

    def close(self):
        self.fileobj.close()

    def get_bytes_sent(self):
        return self._sent

class UploadTask(Task):
    def __init__(self, uploader, container, filename, fileobj):
        super(UploadTask, self).__init__('upload')
        self.uploader = uploader
        self.container = container
        self.filename = filename
        self.fileobj = UploadFileWrapper(fileobj)
        self._data = None

    def execute(self):
        self.fileobj.reset()

        # Under 32 MB, just read the entire file
        if self.fileobj.len < MAX_IN_MEMORY_XFER:
            if self._data is None:
                self._data = self.fileobj.read(self.fileobj.len)
            self.uploader.upload(self.container, self.filename, self._data)
        else:
            self.uploader.upload(self.container, self.filename, self.fileobj)

        # Safely close the file object
        try:
            self.fileobj.close()
        except:
            pass

        return None

    def get_bytes_processed(self):
        return self.fileobj.get_bytes_sent()

    def get_desc(self):
        return 'Upload {}'.format(self.filename)

class PackfileTask(Task):
    def __init__(self, uploader, archive_fs, packfile_type, deid_profile, follow_symlinks, container, filename, paths=None, compression=None):
        super(PackfileTask, self).__init__('packfile')

        self.uploader = uploader
        self.archive_fs = archive_fs
        self.packfile_type = packfile_type
        self.deid_profile = deid_profile
        self.follow_symlinks = follow_symlinks

        self.container = container
        self.filename = filename
        self.paths = paths
        self.compression = compression

        self._bytes_processed = None

    def execute(self):
        tmpfile = tempfile.TemporaryFile()

        create_zip_packfile(tmpfile, self.archive_fs, packfile_type=self.packfile_type, 
            symlinks=self.follow_symlinks, paths=self.paths, compression=self.compression, 
            progress_callback=self.update_bytes_processed, deid_profile=self.deid_profile)

        #Rewind
        tmpfile.seek(0)

        try:
            # Close the filesystem
            archive_fs.close()
        except:
            pass

        # The next task is an uplad task
        return UploadTask(self.uploader, self.container, self.filename, tmpfile)

    def get_bytes_processed(self):
        if self._bytes_processed is None:
            return 0
        return self._bytes_processed

    def get_desc(self):
        return 'Pack {}'.format(self.filename)

    def update_bytes_processed(self, bytes_processed):
        self._bytes_processed = bytes_processed


class UploadQueue(WorkQueue):
    def __init__(self, config, packfile_count=0, upload_count=0, show_progress=True):
        # Detect signed-url upload and start multiple upload threads
        upload_threads = 1
        uploader = config.get_uploader()
        if uploader.supports_signed_url():
            upload_threads = config.concurrent_uploads 

        super(UploadQueue, self).__init__({
            'upload': upload_threads,
            'packfile': config.cpu_count 
        })

        self.uploader = uploader
        self.compression = config.get_compression_type()
        self.follow_symlinks = config.follow_symlinks

        self._progress_thread = None
        if show_progress:
            self._progress_thread = ProgressReporter(self)
            self._progress_thread.log_process_info(config.cpu_count, upload_threads, packfile_count)
            self._progress_thread.add_group('packfile', 'Packing',  packfile_count)
            self._progress_thread.add_group('upload', self.uploader.verb, upload_count + packfile_count)

    def start(self):
        super(UploadQueue, self).start()

        if self._progress_thread:
            self._progress_thread.start()

    def shutdown(self):
        # Shutdown reporting thread
        if self._progress_thread:
            self._progress_thread.shutdown()
            self._progress_thread.final_report()

        super(UploadQueue, self).shutdown()

    def suspend_reporting(self):
        if self._progress_thread:
            self._progress_thread.suspend()

    def resume_reporting(self):
        if self._progress_thread:
            self._progress_thread.resume()

    def log_exception(self, job):
        self.suspend_reporting()

        super(UploadQueue, self).log_exception(job)

        self.resume_reporting()

    def upload(self, container, filename, fileobj):
        self.enqueue(UploadTask(self.uploader, container, filename, fileobj))

    def upload_packfile(self, archive_fs, packfile_type, deid_profile, container, filename, paths=None):
        self.enqueue(PackfileTask(self.uploader, archive_fs, packfile_type, deid_profile, self.follow_symlinks, container, 
            filename, paths=paths, compression=self.compression))

