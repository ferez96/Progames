import logging
from queue import SimpleQueue
from random import randint
from threading import Thread

import pygame

from progames.core.engine import *

# constants
LOG_LEVEL = logging.DEBUG
LOG_FMT = '%(asctime)s - %(levelname)0.1s - [%(threadName)s] - %(name)s - %(message)s'
n = 10

# Visualize
size = width, height = 640, 480
csize = (min(width, height) - 20) // n
font_size = max(20, csize // 4)
padding = 10

BLUE = 0, 0, 255
GREEN = 0, 255, 0
RED = 255, 0, 0
WHITE = 255, 255, 255
BLACK = 0, 0, 0


# commands & events
class Start(Command):
    pass


class Put(Command):
    def __init__(self, x, y, type_):
        self.x, self.y = x, y
        self.type_ = type_


class Spawn(Event):
    def __init__(self, turn, x, y, type_):
        self.turn = turn
        self.x, self.y = x, y
        self.type_ = type_

    def __repr__(self):
        return "#%s: spawn %s at (%s, %s)" % (self.turn, self.type_, self.x, self.y)


class InvalidCommand(Event):
    def __init__(self, turn, x, y, type_):
        self.turn = turn
        self.x, self.y = x, y
        self.type_ = type_

    def __repr__(self):
        return "#%s: can not play at (%s, %s)" % (self.turn, self.x, self.y)


class BeginTurn(Event):
    def __init__(self, turn, board):
        self.turn = turn
        self.board = board

    def __repr__(self):
        return "start turn #%s" % self.turn


class Win(GameEnded):
    def __init__(self, winner, win_chain=None):
        self.winner = winner
        self.win_chain = win_chain

    def __repr__(self):
        return "game ended: %s is the winner" % self.winner


class Draw(GameEnded):
    def __repr__(self):
        return "game ended but there is no winner"


class CaroGame(AbstractGameWorld):
    def __init__(self, size_):
        self.size = size_
        self.started = False
        self.board = None
        self.current_turn = None
        self.cell_played = None
        self.winner = None

    def process(self, command):
        if isinstance(command, Start):
            return self.start()
        if isinstance(command, Put):
            return self.put(command)
        raise RuntimeError("Unknown command")

    def sure(self, x, y):
        return 1 <= x <= self.size and 1 <= y <= self.size

    def start(self):
        if self.started:
            return []
        self.started = True
        self.board = [{} for i in range(self.size + 1)]
        self.current_turn = 1
        self.cell_played = 0
        return [BeginTurn(self.current_turn, self.board)]

    def put(self, cmd):
        x, y, type_ = cmd.x, cmd.y, cmd.type_
        if not self.started:
            return [InvalidCommand(self.current_turn, x, y, type_)]

        if self.sure(x, y) and self.board[x].get(y) is None:
            self.board[x][y] = type_
            self.cell_played += 1
            return self.spawn(x, y, type_)
        else:
            return [InvalidCommand(self.current_turn, x, y, type_)]

    def spawn(self, x, y, type_):
        dx = [0, 1, 1, -1]
        dy = [1, 0, 1, 1]
        for direction in range(4):
            continuous_count = 0
            for i in range(-4, 5):
                xx = x + dx[direction] * i
                yy = y + dy[direction] * i
                if not self.sure(xx, yy) or self.board[xx].get(yy) != type_:  # out board or not continuous
                    continuous_count = 0
                    continue

                continuous_count += 1
                if continuous_count == 5:
                    win_chain = []
                    for ii in range(5):
                        xxx = xx - dx[direction] * ii
                        yyy = yy - dy[direction] * ii
                        win_chain.append((xxx, yyy))
                    self.winner = type_
                    return [Spawn(self.current_turn, x, y, type_), Win(winner=type_, win_chain=win_chain)]

        if self.cell_played < self.size ** 2:
            self.current_turn += 1
            return [Spawn(self.current_turn - 1, x, y, type_), BeginTurn(self.current_turn, self.board)]
        else:
            return [Spawn(self.current_turn, x, y, type_), Draw()]


class CaroProcessor(Processor):
    cps = 10
    p_ = 0

    def handle(self, event):
        super().handle(event)

        if isinstance(event, BeginTurn):
            self.p_ = 1 - self.p_

        GLOBAL_EVENT_QUEUE.put(event)


GLOBAL_EVENT_QUEUE = SimpleQueue()

pygame.init()
screen = pygame.display.set_mode(size)
my_font = pygame.font.SysFont('Comic Sans MS', font_size)

pygame.display.set_caption('Caro Visualization')


def render():
    background = pygame.Surface(screen.get_size())
    background = background.convert()
    background.fill(WHITE)
    screen.blit(background, (0, 0))

    pygame.draw.line(screen, BLACK, (10, 10), (10, 10 + n * csize))
    pygame.draw.line(screen, BLACK, (10, 10), (10 + n * csize, 10))
    pygame.draw.line(screen, BLACK, (10 + n * csize, 10), (10 + n * csize, 10 + n * csize))
    pygame.draw.line(screen, BLACK, (10, 10 + n * csize), (10 + n * csize, 10 + n * csize))
    for i in range(n + 1):
        pygame.draw.line(screen, BLACK, (10, 10 + i * csize), (10 + n * csize, 10 + i * csize))
        pygame.draw.line(screen, BLACK, (10 + i * csize, 10), (10 + i * csize, 10 + n * csize))

    def draw_X(t, l, r, d):
        pygame.draw.line(screen, BLACK, (l, t), (r, d), 5)
        pygame.draw.line(screen, BLACK, (r, t), (l, d), 5)

    def draw_O(t, l, r, d):
        radius = (r - l) // 2
        center = (l + r) // 2, (t + d) // 2
        pygame.draw.circle(screen, BLACK, center, radius, 5)

    running = True
    clock = pygame.time.Clock()
    while running:
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False

        if not GLOBAL_EVENT_QUEUE.empty():
            e = GLOBAL_EVENT_QUEUE.get()
            if isinstance(e, Spawn):
                x, y = e.x, e.y
                l, t = (x - 1) * csize + 10, (y - 1) * csize + 10
                r, d = l + csize, t + csize

                if csize > 4 * padding:
                    t, l = (x + padding for x in (t, l))
                    r, d = (x - padding for x in (r, d))

                if e.type_ == "O":
                    draw_O(t, l, r, d)
                    screen.blit(my_font.render("%3d" % e.turn, True, RED),
                                ((e.x - 1) * csize + 10, (e.y - 1) * csize + 10))
                else:
                    draw_X(t, l, r, d)
                    screen.blit(my_font.render("%3d" % e.turn, True, BLUE),
                                ((e.x - 1) * csize + 10, (e.y - 1) * csize + 10))
            if isinstance(e, Win):
                for (x, y) in e.win_chain:
                    overlay = pygame.Surface((csize, csize))
                    overlay.set_alpha(128)
                    overlay.fill(GREEN)
                    screen.blit(overlay, ((x - 1) * csize + 10, (y - 1) * csize + 10))

        pygame.display.flip()
        clock.tick(60)


if __name__ == "__main__":
    logging.basicConfig(level=LOG_LEVEL, format=LOG_FMT)

    command_store = SimpleQueueStore()
    game = CaroProcessor(CaroGame(n), command_store)
    command_store.store(Start())


    def play():
        while not game.is_over():
            if command_store.empty():
                command_store.store(Put(randint(1, n), randint(1, n), ['X', 'O'][game.p_]))


    dummy_player = Thread(target=play)

    game.start()
    dummy_player.start()
    render()
