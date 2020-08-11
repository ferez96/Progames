from queue import SimpleQueue

from .abc import Store


class SimpleQueueStore(Store):
    def __init__(self):
        self._q = SimpleQueue()
        self._len = 0

    def __len__(self):
        return self._len

    def store(self, entity, *args, **kwargs):
        self._q.put(entity)
        self._len += 1

    def pop(self, *args, **kwargs):
        if self._q.empty():
            return None
        return self._q.get()

    def empty(self):
        return self._q.empty()
