FROM python:3.9-alpine

WORKDIR /load-generator

# Copy requirements first for better caching
COPY requirements.txt .

# Install dependencies
RUN pip install -r requirements.txt

# Copy the load generator script
COPY load_generator.py .

# Make the script executable
RUN chmod +x load_generator.py

# Run the load generator
CMD ["python", "load_generator.py"]