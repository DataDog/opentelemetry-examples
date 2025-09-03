# Use an official Node.js runtime as the base image
FROM node:18-alpine

# Set the working directory
WORKDIR /game_controller

# Copy package files first for better caching
COPY package*.json ./

# Install the project dependencies
RUN npm install

# Copy the rest of the project files
COPY . ./
ENV service_name=controller
ENV logs_exporter=otlp
   

# Expose port 5002 for the app to be accessible
EXPOSE 5002

# Define the command to run the app
CMD [ "node", "controller.js" ]