# Copyright 2025 Datadog, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""The Python implementation of the gRPC route guide server."""

import logging
import signal
import sys
from concurrent import futures

import grpc
from ddtrace import patch_all

# Enable Datadog tracing for gRPC.
# patch_all() automatically instruments grpc client and server calls,
# creating spans with resource names like "grpc.server /routeguide.RouteGuide/GetFeature".
# These spans are sent to the DD Agent (DD_AGENT_HOST:DD_TRACE_AGENT_PORT)
# and appear in Datadog APM with gRPC-specific metadata.
# See: https://ddtrace.readthedocs.io/en/stable/integrations.html#grpc
patch_all()

import route_guide_pb2
import route_guide_pb2_grpc

import route_guide_resources

logger = logging.getLogger(__name__)


def get_feature(feature_db, point):
    """Returns Feature at given location or None."""
    for feature in feature_db:
        if feature.location == point:
            return feature
    return None


class RouteGuideServicer(route_guide_pb2_grpc.RouteGuideServicer):
    """Provides methods that implement functionality of route guide server."""

    def __init__(self):
        self.db = route_guide_resources.read_route_guide_database()

    def GetFeature(self, request, context):
        feature = get_feature(self.db, request)
        if feature is None:
            return route_guide_pb2.Feature(name="", location=request)
        else:
            return feature


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    route_guide_pb2_grpc.add_RouteGuideServicer_to_server(
        RouteGuideServicer(),
        server,
    )
    listen_addr = "0.0.0.0:50051"
    server.add_insecure_port(listen_addr)
    logger.info("Starting server on %s", listen_addr)
    server.start()

    # Graceful shutdown on SIGINT/SIGTERM.
    # Grace period allows in-flight RPCs to complete before forced termination.
    def _shutdown(signum, frame):
        logger.info("Received signal %s, initiating graceful shutdown...", signum)
        # Grace period of 5 seconds for in-flight RPCs to complete.
        done = server.stop(grace=5)
        done.wait()
        logger.info("Server shut down cleanly.")
        sys.exit(0)

    signal.signal(signal.SIGINT, _shutdown)
    signal.signal(signal.SIGTERM, _shutdown)

    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    serve()
