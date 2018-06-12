import tempfile

from abc import ABC, abstractmethod

from .work_queue import Task, WorkQueue
from .packfile import create_zip_packfile
from .progress_reporter import ProgressReporter

class Uploader(ABC):
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

    def execute(self):
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
    def __init__(self, uploader, archive_fs, packfile_type, packfile_args, follow_symlinks, container, filename, paths=None):
        super(PackfileTask, self).__init__('packfile')

        self.uploader = uploader
        self.archive_fs = archive_fs
        self.packfile_type = packfile_type
        self.packfile_args = packfile_args
        self.follow_symlinks = follow_symlinks

        self.container = container
        self.filename = filename
        self.paths = paths

    def execute(self):
        tmpfile = tempfile.TemporaryFile()

        create_zip_packfile(tmpfile, self.archive_fs, packfile_type=self.packfile_type, 
            symlinks=self.follow_symlinks, paths=self.paths, **self.packfile_args)

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
        return 0

    def get_desc(self):
        return 'Pack {}'.format(self.filename)


class UploadQueue(WorkQueue):
    def __init__(self, uploader, packfile_threads=1, upload_threads=1, packfile_count=0, upload_count=0, show_progress=True):
        super(UploadQueue, self).__init__({
            'upload': upload_threads,
            'packfile': packfile_threads
        })

        self.uploader = uploader

        self._progress_thread = None
        if show_progress:
            self._progress_thread = ProgressReporter(self)
            self._progress_thread.add_group('packfile', 'Packing',  packfile_count)
            self._progress_thread.add_group('upload', 'Uploading', upload_count + packfile_count)

    def start(self):
        super(UploadQueue, self).start()

        if self._progress_thread:
            self._progress_thread.start()

    def finish(self):
        self.wait_for_finish()

        # Shutdown reporting thread
        if self._progress_thread:
            self._progress_thread.shutdown()

        # Shutdown
        self.shutdown()

    def upload(self, container, filename, fileobj):
        self.enqueue(UploadTask(self.uploader, container, filename, fileobj))

    def upload_packfile(self, archive_fs, packfile_type, packfile_args, follow_symlinks, container, filename, paths=None):
        self.enqueue(PackfileTask(self.uploader, archive_fs, packfile_type, packfile_args, follow_symlinks, container, filename, paths=paths))

