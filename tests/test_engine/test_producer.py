from progames.core.engine.producers import Producer


def test_producer(store):
    async def gen_func():
        for i in range(10):
            yield i

    producer = Producer(store, gen_func())
    producer.start()
    assert 10 == len(store)
