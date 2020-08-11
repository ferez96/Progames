__all__ = ["Producer"]

import asyncio
from collections.abc import AsyncGenerator

from . import Store, concurrent


class Producer:
    def __init__(self, store: Store, generator: AsyncGenerator):
        self.store = store
        self.generator = generator

    async def produce_async(self):
        async for command in self.generator:
            self.store.store(command)

    def start(self):
        """listen to inputs"""

        def task():
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            loop.run_until_complete(self.produce_async())

        concurrent.run(task)
