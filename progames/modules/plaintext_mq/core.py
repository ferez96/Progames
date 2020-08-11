from .abc import Subscriber
from typing import Dict, Any, List, Set


class Broker:
    __instance: "Broker" = None
    __channels: Dict[str, Set[Subscriber]] = {}
    __subscriptions: Dict[int, int] = {}
    __message_queues: Any = None  # no queue yet

    def __init__(self):
        if Broker.__instance is not None:
            raise Exception("This class is a singleton!")
        else:
            Broker.__instance = self

    @staticmethod
    def get_instance():
        if Broker.__instance is None:
            Broker()
        return Broker.__instance

    @staticmethod
    def hash(obj):
        """hash stuffs"""
        return hash(obj)

    def subscribe(self, channel: str, subscriber):
        if isinstance(subscriber, Subscriber):
            subscribe_key = self.hash(subscriber)

            if channel not in self.__channels:
                self.__channels[channel] = set()  # new channel
            if subscribe_key not in self.__subscriptions:
                self.__subscriptions[subscribe_key] = 0  # new subscriber

            # if only the subscriber not subscribe the channel yet
            if subscriber not in self.__channels[channel]:
                self.__subscriptions[subscribe_key] += 1
                self.__channels[channel].add(subscriber)
        else:
            raise TypeError("subscriber is not implement interface Subscriber")

    def unsubscribe(self, channel, subscriber):
        subscribe_key = self.hash(subscriber)

        if subscribe_key not in self.__subscriptions:
            raise Exception("subscriber did not subscribe this channel")

        self.__channels[channel].remove(subscriber)

        self.__subscriptions[subscribe_key] -= 1
        if self.__subscriptions[subscribe_key] == 0:
            del self.__subscriptions[subscribe_key]

    def broadcast(self, message):
        receivers = set()

        for _, subscribers in self.__channels.items():
            for subscriber in subscribers:
                receivers.add(subscriber)

        for receiver in receivers:
            self.deliver(receiver, message)  # async

        return len(receivers)

    def publish(self, channel, message):
        if channel not in self.__channels:
            return 0

        for receiver in self.__channels[channel]:
            self.deliver(receiver, message)  # async

        return len(self.__channels[channel])

    @staticmethod
    def deliver(receiver, message):
        receiver.receive(message)


def subscribe(channel, subscriber):
    Broker.get_instance().subscribe(channel, subscriber)


def unsubscribe(channel, subscriber):
    Broker.get_instance().unsubscribe(channel, subscriber)


def broadcast(message):
    Broker.get_instance().broadcast(message)


def publish(channel, message):
    Broker.get_instance().publish(channel, message)
