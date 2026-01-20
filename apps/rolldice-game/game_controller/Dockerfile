# Use an official Node.js runtime as the base image
FROM node:14-alpine

# Set the working directory
WORKDIR /game_controller

# Copy the project files to the working directory
COPY . ./game_controller
COPY package*.json ./game_controller/

WORKDIR /game_controller/game_controller

# Install the project dependencies
RUN npm install
ENV service_name=controller
ENV logs_exporter=otlp
   

# Expose port 5002 for the app to be accessible
EXPOSE 5002

# Define the command to run the app
CMD [ "node", "controller.js" ]