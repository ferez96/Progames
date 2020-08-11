import pytest

from progames.core.engine import AbstractGameWorld, Command, Store


def test_processor__process_frame(mock_processor):
    mock_processor.process_command()
    assert 1 == mock_processor.frame_count


def test_processor__process_frame__wrong_command(mock_processor):
    class FailCommandStore(Store):
        def empty(self):
            pass

        def __len__(self) -> int:
            pass

        def __init__(self):
            self.called = 0

        def store(self, entity, *args, **kwargs):
            pass

        def pop(self, *args, **kwargs):
            self.called += 1
            return "This is completely fail"

    mock_processor.command_store = FailCommandStore()
    mock_processor.process_command()
    assert 1 == mock_processor.frame_count
    assert 1 == mock_processor.command_store.called


def test_processor__process_frame__get_next_command_raise_error(mock_processor):
    class RaiseErrorCommandStore(Store):
        def empty(self):
            return True

        def __len__(self) -> int:
            return 0

        def store(self, entity, *args, **kwargs):
            raise Exception("Can not store")

        def pop(self, *args, **kwargs):
            raise Exception("No command found")

    mock_processor.command_store = RaiseErrorCommandStore()
    with pytest.raises(RuntimeWarning):
        mock_processor.process_command()


def test_processor__start__update_game_world_fail(mock_processor):
    class FailWorld(AbstractGameWorld):
        def process(self, command) -> "list[Event]":
            raise Exception("tiditada")

    mock_processor.state = FailWorld()
    mock_processor.command_store.store(Command())
    mock_processor.process_command()


def test_processor__start(mock_processor):
    mock_processor.start()
    mock_processor.over = True
