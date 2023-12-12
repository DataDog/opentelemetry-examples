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

    @app.route("/users", methods=["GET"])
    def get_users():
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("SELECT * FROM users")
            users = cur.fetchall()
        return jsonify(users)

    @app.route("/user/<int:user_id>", methods=["GET"])
    def get_user(user_id):
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("select * from users where id = ?", user_id)
            user = cur.fetchone()
        if user:
            return jsonify(user)
        return jsonify({"error": "User not found"}), 404

    @app.route("/user", methods=["POST"])
    def create_user():
        if not request.json:
            return jsonify({"error": "JSON payload expected"}), 400
        name = request.json.get("name")
        email = request.json.get("email")
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("INSERT INTO users (name, email) VALUES (?, ?)", (name, email))
            conn.commit()
        return jsonify({"message": "User created successfully"}), 201

    @app.route("/user/<int:user_id>", methods=["PUT"])
    def update_user(user_id):
        if not request.json:
            return jsonify({"error": "JSON payload expected"}), 400
        name = request.json.get("name")
        email = request.json.get("email")
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute(
                "UPDATE users SET name=?, email=? WHERE id=?", (name, email, user_id)
            )
            conn.commit()
        return jsonify({"message": "User updated successfully"})

    @app.route("/user/<int:user_id>", methods=["DELETE"])
    def delete_user(user_id):
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("DELETE FROM users WHERE id=?", (user_id,))
            conn.commit()
        return jsonify({"message": "User deleted successfully"})

    @app.route("/user/name/<string:name>", methods=["GET"])
    def get_user_by_name(name):
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("SELECT * FROM users WHERE name=?", (name,))
            user = cur.fetchone()
        if user:
            return jsonify(user)
        return jsonify({"error": "User not found"}), 404

    @app.route("/select", methods=["GET"])
    def select_user():
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("SELECT * FROM users where id = 1")
            users = cur.fetchall()
        return jsonify(users)

    return app


if __name__ == "__main__":
    app = create_app()
    app.run(debug=False, host="0.0.0.0", port=os.environ.get("SERVER_PORT", 9090))
