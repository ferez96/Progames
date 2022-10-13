"""
Command Line Interface
"""
import sys

from click import Group, option, argument

cli = Group(help="""Welcome to Progames""")
game = Group()
cli.add_command(game, "game")



@cli.command()
def start():
    """Start http server"""
    from progames.server import create_app
    create_app().run("localhost", 3000)


@game.command("install")
@argument("uri")
def _import(uri):
    """Import game"""
    print("install game from:", uri)


def main():
    cli.main(args=sys.argv[1:])
