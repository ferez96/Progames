from flask import Blueprint, render_template


bp = Blueprint("admin", __name__, url_prefix="/admin")


@bp.route("", methods=["GET"])
def index():
    return render_template('admin/index.html')


@bp.route("/users", methods=["GET"])
def users():
    return render_template('admin/users.html')