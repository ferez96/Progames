__all__ = ["Store", "GameWorld", "Message"]

import abc
from collections.abc import Sized
from typing import TypeVar, Iterable

Item = TypeVar('Item')


class Store(Sized, metaclass=abc.ABCMeta):
    @abc.abstractmethod
    def store(self, entity: Item, *args, **kwargs) -> None:
        """store item

        Args:
            entity: item to store
            """
        pass

    @abc.abstractmethod
    def pop(self, *args, **kwargs) -> Item:
        """get the first item

        Returns:
            the first item
            """
        pass

    @abc.abstractmethod
    def empty(self) -> bool:
        """check if the store is empty"""
        pass


class Message(metaclass=abc.ABCMeta):
    # TODO: Support serialize & deserialize
    pass


class GameWorld(metaclass=abc.ABCMeta):
    @abc.abstractmethod
    def accept(self, command: "progames.core.engine.messages.Command") \
            -> Iterable["progames.core.engine.messages.Event"]:
        """
        accept

        Args:
            command: command
        """
        pass

    @abc.abstractmethod
    def update(self) -> None:
        """
        Some game need update game world each frame
        """
        pass

    @abc.abstractmethod
    def rollback(self, command: "progames.core.engine.messages.Command") -> None:
        """
        rollback

        Args:
            command: the failed command which causes rollback
        """
        pass
