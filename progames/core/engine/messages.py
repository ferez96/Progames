from typing import Iterable

from progames.core.engine.abc import GameWorld, Message


class Command(Message):
    """base command"""

    def visit(self, world: GameWorld) -> Iterable['Event']:
        if not isinstance(world, GameWorld):
            raise RuntimeError("can not visit %s" % world)
        return world.accept(self)


class Event(Message):
    """base event"""
    pass


class GameEnded(Event):
    """default event raise when game is ended"""
    pass
