import collections
import sys
import threading

import fs.filesize

from datetime import datetime
from abc import ABC, abstractmethod

class GroupStats(object):
    def __init__(self, desc, total_count):
        self.desc = desc
        self.total_count = total_count
        self.completed = 0
        self.completed_bytes = 0

        self.samples = collections.deque()
        self.bytes_per_sec = 0

class ProgressReporter(ABC):
    """Thread that prints upload progress"""
    def __init__(self, queue, average_samples=10, sample_time=0.5, columns=80):
        self.queue = queue

        self.groups = collections.OrderedDict()

        self.completed_bytes = 0
        self.total_bytes = 0
        self.uploaded_files = 0
        self.pending_filename = ''

        self.sample_time = sample_time
        self.average_samples = average_samples
        self.columns = columns

        self._suspended = False
        self._running = False
        self._thread = None
        self._shutdown_event = threading.Event()

    def add_group(self, name, desc, total_count):
        self.groups[name] = GroupStats(desc, total_count)

    def start(self):
        self._running = True
        self._suspended = False
        self._thread = threading.Thread(target=self.run, name='progress-report-thread')
        self._thread.start()

    def suspend(self):
        self._suspended = True

    def resume(self):
        self._suspended = False

    def shutdown(self):
        self._running = False
        self._shutdown_event.set()
        self._thread.join()

    def run(self):
        while True:
            self._shutdown_event.wait(self.sample_time)
            if not self._running:
                return

            self.sample()
            if not self._suspended:
                self.report()

    def sample(self):
        # Get current stats
        current_stats = self.queue.get_stats()
        sample_time = datetime.now()

        for group, stats in current_stats.items():
            # Take a sample from each group for averaging
            group_stats = self.groups[group]
            group_stats.completed = stats.get('completed', 0)
            group_stats.completed_bytes = stats.get('completed_bytes', 0)
            group_stats.samples.append((sample_time, group_stats.completed_bytes))

            # Prune older samples
            while len(group_stats.samples) > self.average_samples:
                group_stats.samples.popleft()

            # Calculate the average between the oldest and newest
            if len(group_stats.samples) > 1:
                t1, s1 = group_stats.samples[0]
                t2, s2 = group_stats.samples[-1]

                dt = (t2 - t1).total_seconds()
                ds = s2 - s1

                group_stats.bytes_per_sec = (s2 - s1) / dt

    def report(self):
        messages = []

        for group in self.groups.values():
            if group.total_count:
                if group.completed == group.total_count:
                    bps = 'DONE'
                else:
                    bps = fs.filesize.traditional(group.bytes_per_sec) + '/s'
                messages.append('{} {}/{} - {}'.format(group.desc, group.completed, group.total_count, bps))

        message = ', '.join(messages).ljust(self.columns) + '\r'

        sys.stdout.write(message)
        sys.stdout.flush()

