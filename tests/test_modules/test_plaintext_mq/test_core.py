import pytest
import threading

from progames.modules.plaintext_mq.core import Broker


@pytest.fixture()
def subscribers():
    class Subscriber:
        received_count = 0
        received = None

        def receive(self, message):
            self.received_count += 1
            self.received = message

        def reset_mock(self):
            self.received_count = 0
            self.received = None

    subs = [Subscriber() for _ in range(10)]
    yield subs
    for sub in subs:
        sub.reset_mock()


def test_subscribe(subscribers):
    subs_1, subs_21, subs_22, subs_3 = subscribers[:4]
    broker = Broker.get_instance()

    broker.subscribe("channel-1", subs_1)
    broker.subscribe("channel-2", subs_21)
    broker.subscribe("channel-2", subs_22)
    broker.subscribe("channel-3", subs_3)

    assert 1 == broker.publish("channel-1", "Hello World")
    assert 1 == subs_1.received_count
    assert 0 == subs_21.received_count
    assert 0 == subs_22.received_count
    assert 0 == subs_3.received_count
    assert "Hello World" == subs_1.received

    assert 2 == broker.publish("channel-2", "World Hello")
    assert 1 == subs_1.received_count
    assert 1 == subs_21.received_count
    assert 1 == subs_22.received_count
    assert 0 == subs_3.received_count
    assert "World Hello" == subs_21.received
    assert "World Hello" == subs_22.received

    assert 1 == broker.publish("channel-3", "Lorem Ipsum")
    assert 1 == subs_1.received_count
    assert 1 == subs_21.received_count
    assert 1 == subs_22.received_count
    assert 1 == subs_3.received_count
    assert "Lorem Ipsum" == subs_3.received

    assert 4 == broker.broadcast("Broadcast")
    assert 2 == subs_1.received_count
    assert 2 == subs_21.received_count
    assert 2 == subs_22.received_count
    assert 2 == subs_3.received_count
    assert "Broadcast" == subs_1.received
    assert "Broadcast" == subs_21.received
    assert "Broadcast" == subs_22.received
    assert "Broadcast" == subs_3.received

    broker.subscribe("channel-1", subs_21)
    assert 2 == broker.publish("channel-1", "New Message")
    assert 3 == subs_1.received_count
    assert 3 == subs_21.received_count
    assert 2 == subs_22.received_count
    assert 2 == subs_3.received_count
    assert "New Message" == subs_1.received
    assert "New Message" == subs_21.received

    assert 4 == broker.broadcast("New Broadcast")
    assert 4 == subs_1.received_count
    assert 4 == subs_21.received_count
    assert 3 == subs_22.received_count
    assert 3 == subs_3.received_count
    assert "New Broadcast" == subs_1.received
    assert "New Broadcast" == subs_21.received
    assert "New Broadcast" == subs_22.received
    assert "New Broadcast" == subs_3.received


def test_publish_multi_threads():
    import time

    class AsyncSubscriber:
        received_count = 0
        received = None

        def receive(self, message):
            time.sleep(0.1)
            self.received_count += 1
            self.received = message

        def reset_mock(self):
            self.received_count = 0
            self.received = None

    class Future:
        instance = None

        def set(self, instance):
            self.instance = instance

    def _subscribe(future):
        broker = Broker.get_instance()
        subscriber = AsyncSubscriber()
        broker.subscribe("async-channel", subscriber)
        future.set(subscriber)

    def _publish():
        broker = Broker.get_instance()
        time.sleep(0.2)
        return broker.publish("async-channel", "message")

    future = Future()
    thread = threading.Thread(target=_subscribe, args=(future,))
    assert 0 == _publish()
    thread.start()
    thread.join()
    assert future.instance is not None
    subscriber = future.instance
    assert 1 == _publish()
    assert 1 == subscriber.received_count
    threading.Thread(target=_publish).start()
    threading.Thread(target=_publish).start()
    time.sleep(0.35)
    assert 3 == subscriber.received_count


def test_publish_unknown_channel(subscribers):
    broker = Broker.get_instance()
    assert 0 == broker.publish("channel_1", "Anything")


def test_unsubscribe(subscribers):
    broker = Broker.get_instance()
    broker.subscribe("vtv3", subscribers[0])
    broker.subscribe("htv7", subscribers[0])
    broker.subscribe("htv7", subscribers[1])

    broker.unsubscribe("vtv3", subscribers[0])

    assert 0 == broker.publish("vtv3", "news")
    assert 2 == broker.publish("htv7", "cartoon")

    assert 1 == subscribers[0].received_count
    assert 1 == subscribers[1].received_count
    assert "cartoon" == subscribers[0].received
    assert "cartoon" == subscribers[1].received
