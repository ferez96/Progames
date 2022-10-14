from progames.server.db import get_db

SELECT_USER_BY_ID = 'SELECT * FROM user WHERE id = ?'
SELECT_USER_BY_USERNAME = 'SELECT * FROM user WHERE username = ?'


def get_user_by_id(user_id):
    return get_db().execute(
            SELECT_USER_BY_ID, (user_id,)
        ).fetchone()


def get_user_by_username(username):
    return get_db().execute(
        SELECT_USER_BY_USERNAME, (username,)
    ).fetchone()
