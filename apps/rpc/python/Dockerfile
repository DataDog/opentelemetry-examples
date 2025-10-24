FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy proto file and generate gRPC code
COPY protos/route_guide.proto .
RUN python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. route_guide.proto

# Copy application files
COPY route_guide_server.py .
COPY route_guide_client.py .
COPY route_guide_resources.py .
COPY route_guide_db.json .

# Expose the gRPC port
EXPOSE 50051

# Default command (will be overridden in docker-compose)
CMD ["python", "route_guide_server.py"]
