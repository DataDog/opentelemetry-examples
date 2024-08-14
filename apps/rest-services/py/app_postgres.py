import os

import psycopg2
from flask import Flask, jsonify, request

DATABASE = "dbname=postgres user=otel.sample host=localhost port=5432"


def get_db():
    conn = psycopg2.connect(DATABASE)
    return conn


def setup_database():
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS users (
                    id SERIAL PRIMARY KEY,
                    name TEXT NOT NULL,
                    email TEXT NOT NULL UNIQUE
                );
            """
            )


def create_app():
    app = Flask(__name__)
    setup_database()

    @app.route("/users", methods=["GET"])
    def get_users():
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("SELECT id, name, email FROM users")
            users = cur.fetchall()
        return jsonify(users)

    @app.route("/user/<int:user_id>", methods=["GET"])
    def get_user(user_id):
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("SELECT id, name, email FROM users WHERE id = %s", (user_id,))
            user = cur.fetchone()
        if user:
            return jsonify(user)
        return jsonify({"error": "User not found"}), 404

    @app.route("/user/name/<string:name>", methods=["GET"])
    def get_user_by_name(name):
        with get_db() as conn:
            cur = conn.cursor()
            cur.execute("SELECT id, name, email FROM users WHERE name = %s", (name,))
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
            with conn.cursor() as cur:
                cur.execute(
                    "INSERT INTO users (name, email) VALUES (%s, %s) RETURNING id;",
                    (name, email),
                )
                user_id = cur.fetchone()[0]
                conn.commit()

        return (
            jsonify({"message": "User created successfully", "user_id": user_id}),
            201,
        )

    @app.route("/user/<int:user_id>", methods=["PUT"])
    def update_user(user_id):
        if not request.json:
            return jsonify({"error": "JSON payload expected"}), 400

        name = request.json.get("name")
        email = request.json.get("email")

        with get_db() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "UPDATE users SET name=%s, email=%s WHERE id=%s",
                    (name, email, user_id),
                )
                if cur.rowcount == 0:
                    return jsonify({"error": "User not found"}), 404
                conn.commit()

        return jsonify({"message": "User updated successfully"}), 200

    @app.route("/user/<int:user_id>", methods=["DELETE"])
    def delete_user(user_id):
        with get_db() as conn:
            with conn.cursor() as cur:
                cur.execute("DELETE FROM users WHERE id=%s", (user_id,))
                if cur.rowcount == 0:
                    return jsonify({"error": "User not found"}), 404
                conn.commit()

        return jsonify({"message": "User deleted successfully"}), 200

    return app


if __name__ == "__main__":
    app = create_app()
    app.run(debug=False, host="0.0.0.0", port=os.environ.get("SERVER_PORT", 9090))
