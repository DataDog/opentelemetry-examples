import os
import sqlite3

from flask import Flask, jsonify, request

DATABASE = "users.db"


def setup_database():
    with get_db() as conn:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS users (
                id INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                email TEXT NOT NULL UNIQUE
            );
        """
        )


def get_db():
    return sqlite3.connect(DATABASE)


def create_app(test_config=None):
    app = Flask(__name__)
    setup_database()

    @app.route("/user/<int:user_id>", methods=["GET"])
    def get_user(user_id):
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("select * from users where id = ?", user_id)
            user = cur.fetchone()
        if user:
            return jsonify(user)
        return jsonify({"error": "User not found"}), 404

    return app


if __name__ == "__main__":
    app = create_app()
    app.run(debug=False, host="0.0.0.0", port=os.environ.get("SERVER_PORT", 9090))
