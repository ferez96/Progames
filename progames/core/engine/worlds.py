import abc
from typing import List

from .abc import GameWorld


class AbstractGameWorld(GameWorld):
    def update(self) -> None:
        pass

    def rollback(self, command: "progames.core.engine.messages.Command") -> None:
        pass

    def accept(self, command: "progames.core.engine.messages.Command") -> List["progames.core.engine.messages.Event"]:
        self.validate(command)
        return self.process(command)

    def validate(self, command: "progames.core.engine.messages.Command"):
        """validate command

        Raises:
            ValidationError: invalid command
        """
        pass

    @abc.abstractmethod
    def process(self, command: "progames.core.engine.messages.Command") -> List["progames.core.engine.messages.Event"]:
        """put your logic here

        Args:
            command: the command

        Returns:
            list of events

        Raises:
            RuntimeError: game crashed
            RuntimeWarning: continue
        """
        pass
