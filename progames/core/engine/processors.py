__all__ = ["Processor"]

import logging
import time

from .messages import GameEnded
from ..exceptions import ValidationError


class Processor:
    """Process game and produce events

    Notes:
        - Player's input data will be processed and transform to commands.
        - Each frame, game only consume 1 command from command store.
        - Command is a visitor, travel around game world and produce events.
          These events will be collected and sent to listeners.
        - Rollback is required to implement
    """

    command_store: 'progames.core.engine.Store' = None  # required
    """command store"""

    state: 'progames.core.engine.GameWorld' = None  # required

    cps: float = 50
    """commands per seconds (max), 0 < cps"""

    over: bool
    """flag for checking if game is over"""

    def __init__(self,
                 game: 'progames.core.engine.GameWorld',
                 store: 'progames.core.engine.Store',
                 *args, **kwargs) -> None:
        """Create a processor for game, using command store"""
        self.state = game
        self.command_store = store
        self.over = False

    def stop(self) -> None:
        """immediate stop game loop"""
        self.over = True

    def is_over(self) -> bool:
        """check if game is over"""
        return self.over

    def handle(self, event: 'progames.core.engine.Event') -> None:
        """base handle event, don't forget call: `super().handle(event)`"""
        if isinstance(event, GameEnded):
            self.stop()

    def get_next_command(self) -> 'typing.Optional[progames.core.engine.Command]':
        """get command to process

        :raise RuntimeWarning
        """
        return self.command_store.pop()

    def process_command(self) -> None:
        """process commands one by one

        Flows:
            1. pop 1 command from command store
            2. the command go through game world, make changes and produce events
            3. notify events to listeners (register)
            4. update game world

        :raise RuntimeError: game must stop
        :raise RuntimeWarning: game continue
        """
        try:
            command = self.get_next_command()
        except Exception as e:
            raise RuntimeWarning("Can not get next command") from e
        if command:
            try:
                events = command.visit(self.state)
                for event in events:
                    self.handle(event)
            except ValidationError as e:  # invalid command
                logging.warning("invalid command", exc_info=True)
            except Exception as e:
                try:
                    self.state.rollback(command)  # may fail
                    logging.warning("process command failed", exc_info=True)
                except Exception as e:
                    raise RuntimeError("game can not roll back") from e

    def start(self):
        """start processor"""

        def _process():
            while not self.is_over():
                self.process_command()
                try:
                    self.state.update()
                except Exception as e:
                    raise RuntimeError("new state can not be created") from e
                current = time.time()
                delay = max((int(current * self.cps) + 1) / self.cps - current, 0)
                time.sleep(delay)  # sleep to next command

        from threading import Thread
        Thread(target=_process).start()
