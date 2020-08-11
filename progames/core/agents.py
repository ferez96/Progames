import io
import abc


class AbstractPlayer(abc.ABC):
    """Abstract class for players, give basic functions which are required by flow"""

    @abc.abstractmethod
    def get_input_stream(self) -> io.IOBase:
        pass

    @abc.abstractmethod
    def get_output_stream(self) -> io.IOBase:
        pass

    @abc.abstractmethod
    def start(self):
        pass

    @abc.abstractmethod
    def stop(self):
        pass
