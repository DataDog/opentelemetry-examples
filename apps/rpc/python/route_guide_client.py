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
"""The Python implementation of the gRPC route guide client."""

import logging

import grpc
from ddtrace import patch_all

# Enable Datadog tracing for gRPC.
# patch_all() automatically instruments grpc channels, creating client spans
# with resource names like "grpc.client /routeguide.RouteGuide/GetFeature".
# Context propagation (trace ID, parent span ID) is injected into gRPC
# metadata automatically, enabling distributed tracing across services.
# See: https://ddtrace.readthedocs.io/en/stable/integrations.html#grpc
patch_all()

import route_guide_pb2
import route_guide_pb2_grpc

logger = logging.getLogger(__name__)


def format_point(point):
    # Not delegating in point.__str__ because it is an empty string when its
    # values are zero. In addition, it puts a newline between the fields.
    return f"latitude: {point.latitude}, longitude: {point.longitude}"


def run():
    """
    The client makes a simple unary RPC call by calling RPC method GetFeature.
    The ddtrace library automatically creates a client span and propagates
    the trace context to the server via gRPC metadata headers.
    """
    channel = grpc.insecure_channel("grpc-server:50051")
    stub = route_guide_pb2_grpc.RouteGuideStub(channel)
    point = route_guide_pb2.Point(latitude=412346009, longitude=-744026814)
    try:
        feature = stub.GetFeature(point)
        if feature.name:
            print(f"Feature called '{feature.name}' at {format_point(feature.location)}")
        else:
            print(f"Found no feature at {format_point(feature.location)}")
    except grpc.RpcError as e:
        # Log gRPC status code and details for better error diagnostics.
        # The ddtrace library will automatically tag the span with error info.
        logger.error(
            "gRPC call failed: code=%s details=%s",
            e.code(),
            e.details(),
        )
        raise


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    run()
