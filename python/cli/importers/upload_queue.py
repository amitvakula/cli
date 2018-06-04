from abc import ABC, abstractmethod

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

class SynchronousUploadQueue(object):
    """Simplest upload queue, performs uploads synchronously"""
    def __init__(self, uploader):
        self.uploader = uploader

    def upload(self, container, name, fileobj):
        """Queue the given file for uploading, closing fileobj after upload completes.

        Arguments:
            container (ContainerNode): The destination container
            name (str): The file name
            fileobj (obj): The file-like object, which supports read(), and (optionally) close()
        """
        try:
            self.uploader.upload(container, name, fileobj)
        finally:
            if hasattr(fileobj, 'close'):
                fileobj.close()

    def finish(self):
        """Wait for all queued uploads to finish uploading"""
        pass

