from abc import ABCMeta, ABC, abstractmethod


class Subscriber(metaclass=ABCMeta):
    @classmethod
    def __subclasshook__(cls, sub_cls):
        if cls is Subscriber:
            if any("receive" in B.__dict__ for B in sub_cls.__mro__):
                return True
        return NotImplemented

    @abstractmethod
    def receive(self, *args, **kwargs):
        pass
