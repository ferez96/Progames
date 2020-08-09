"""
Command Line Interface
"""
import sys

from click import Group, option, argument

cli = Group(help="""Welcome to Progames""")
match = Group()
cli.add_command(match, "match")


@match.command()
@argument("game")
def start(game):
    """Start new match"""
    print("Mock", game)


@match.command()
@argument("file")
def visualize(file):
    """Visualize match"""
    print(file)


def main():
    cli.main(args=sys.argv[1:])
