import sqlite3

import click
from flask import current_app, g
from werkzeug.security import generate_password_hash
from getpass import getpass

def get_db():
    if 'db' not in g:
        g.db = sqlite3.connect(
            current_app.config['DATABASE'],
            detect_types=sqlite3.PARSE_DECLTYPES
        )
        g.db.row_factory = sqlite3.Row

    return g.db


def close_db(e=None):
    db = g.pop('db', None)

    if db is not None:
        db.close()


def init_db():
    db = get_db()
    INSERT_USER_SQL = "INSERT INTO user (`username`, `password`) VALUES ('{username}', '{password}')"

    # schema
    with current_app.open_resource('sql/schema.sql') as f:
        db.executescript(f.read().decode('utf8'))

    # add root user
    username = input("Enter root admin username (root):") or "root"
    password = getpass("Enter password (admin):") or "admin"
    password_hash = generate_password_hash(password)
    db.executescript(INSERT_USER_SQL.format(username=username, password=password_hash))


@click.command('init-db')
def init_db_command():
    """Clear the existing data and create new tables."""
    init_db()
    click.echo('Initialized the database.')


def init_app(app):
    app.teardown_appcontext(close_db)
    app.cli.add_command(init_db_command)
