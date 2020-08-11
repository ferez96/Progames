from progames.core.engine import *

# Game cá ngựa
# 4 nguoi choi
# moi nguoi choi co 4 ngua
# 2 vien xuc xac
# neu nhu roll 2 vien cung so thi co the ra ngua
# neu nhu roll 2 vien cung 1 hoac cung 6 thi duoc di tiep
# con lai thi di (neu duoc)
#   neu tu current_pos -> dest trong => di duoc
#   neu current_pos -> dest-1 trong + dest co horse of other player => kick
# di het vong coi nhu ve dich


# Command - Player sinh ra
#   + SetHorse
#   + MoveHorse

# Event - Game sinh ra
#   - GameEnded <- mac dinh cua he thong
#   + StartGame
#   + StartTurn
#   + HorseSetted - xuat quan
#   + HorseMoved - di chuyen
#   + HorseKick - da ngua doi phuong
#   + HorseFinish - ve dich


def render(horses):
    temp = ["." for _ in range(12*4)]
    for i in range(len(horses)):
        if horses[i].pos is not None:
            temp[horses[i].pos] = str(i)

    print("".join(temp))


class SetHorse(Command):
    def __init__(self, horse_id, player_id):
        self.horse_id = horse_id
        self.player_id = player_id


class MoveHorse(Command):
    def __init__(self, horse_id, distance):
        self.horse_id = horse_id
        self.distance = distance

class EndGame(Command):
    pass


class HorseKick(Event):
    pass


class Horse:
    def __init__(self, player):
        self.pos = None  # trong chuong
        self.player = player
        self.length = 0

    def __repr__(self):
        return "%s's horse: %s" % ("ABCD"[self.player], self.pos)


class Game(AbstractGameWorld):
    def __init__(self):
        self.current_turn = 1
        self.horses = [Horse(player=i//4) for i in range(16)]


    def process(self, command) -> "list[Event]":
        render(self.horses)

        if isinstance(command, EndGame):
            return [GameEnded()]
        else:
            self.current_turn += 1
            if isinstance(command, SetHorse):
                horse = self.horses[command.horse_id]
                player_id = command.player_id
                horse.pos = player_id * 12
                print("set horse", horse)
            elif isinstance(command, MoveHorse):
                horse = self.horses[command.horse_id]
                current_pos = horse.pos
                if current_pos is None:
                    print("ngua chua xuat phat")
                    return []

                dest = horse.pos + command.distance
                # finish_point = horse.player*12

                for i in range(16):
                    other_horse = self.horses[i]
                    # ngua 1: 3
                    # ngua 2: 45
                    # move: 2 - 7 => 45 -3-> 0 -4-> 4
                    # neu co ngua o tren duong
                    if other_horse.pos is not None:
                        if dest >= 48:
                            dest_ = dest % 48
                            if other_horse.pos < dest_ or other_horse.pos > current_pos:
                                print("Khong di duoc")
                                return []
                        elif current_pos < other_horse.pos < dest:
                            print("Khong di duoc")
                            return []

                # di duoc
                for i in range(16):
                    other_horse = self.horses[i]
                    # kick
                    if other_horse.pos is not None and other_horse.pos % 48 == dest % 48\
                        and other_horse.player != horse.player:
                        print("Da ngua %s" % i)
                        other_horse.pos = None

                        self.move(horse, dest)
                        return [HorseKick()]

                print("move horse %s a distance of %s"%(command.horse_id, command.distance))
                self.move(horse, dest)
            return []

    def move(self, horse, dest):
        if horse.length >= 12*4:
            print("horse finished")
            horse.pos = None
            return
        else:
            print("hore move")
            horse.length += dest - horse.pos
            horse.pos = dest % 48


class MyProcessor(Processor):
    pass


if __name__ == "__main__":
    command_store = SimpleQueueStore()
    processor = MyProcessor(Game(), command_store)
    processor.start()

    command_store.store(SetHorse(0, 0))  # player 0 set horse 0
    command_store.store(SetHorse(4, 1))  # player 1 set horse 0 (id=4)
    command_store.store(SetHorse(8, 2))  # player 2 set horse 1 (id=8)
    command_store.store(MoveHorse(0, 6))
    command_store.store(MoveHorse(0, 6))
    command_store.store(SetHorse(3, 0))
    command_store.store(MoveHorse(0, 12))
    command_store.store(MoveHorse(0, 12))
    command_store.store(MoveHorse(0, 12))
    command_store.store(SetHorse(8, 2))  # player 2 set horse 1 (id=8)
    command_store.store(MoveHorse(8, 12))
    command_store.store(MoveHorse(8, 12))
    command_store.store(MoveHorse(8, 12))
    command_store.store(MoveHorse(8, 12))
    command_store.store(EndGame())  # just for game end

