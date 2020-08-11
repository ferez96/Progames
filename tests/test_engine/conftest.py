import pytest

from progames.core.engine import AbstractGameWorld, SimpleQueueStore, Processor


class MockWorld(AbstractGameWorld):
    def process(self, command) -> "list[Event]":
        return []


class MockProcessor(Processor):
    cps = 5

    def handle(self, event):
        super().handle(event)

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.frame_count = 0
        self.over = False

    def process_command(self):
        super().process_command()
        self.frame_count += 1

    def is_over(self, *args, **kwargs):
        return self.over


@pytest.fixture()
def store():
    return SimpleQueueStore()


@pytest.fixture()
def mock_processor(store):
    processor = MockProcessor(MockWorld(), store)
    yield processor
