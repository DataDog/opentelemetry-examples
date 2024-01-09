# Unless explicitly stated otherwise all files in this repository are dual-licensed
# under the Apache 2.0 or BSD3 Licenses.
#
# This product includes software developed at Datadog (https://www.datadoghq.com/)
# Copyright 2022 Datadog, Inc.
from datetime import datetime, timedelta
from random import randint
from flask import Flask, request
import os

app = Flask(__name__)


@app.route("/")
def index():
    return "Welcome to the calendar app!"


@app.route("/server_request")
def server_request():
    return "served"


@app.route("/calendar")
def get_date():
    """Generates a random date in 2022."""

    day_offset = randint(0, 365)
    start_date = datetime(2022, 1, 1)
    output = start_date + timedelta(days=day_offset)
    return output.strftime("%m/%d/%Y")


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=os.environ.get('SERVER_PORT', 9090))
