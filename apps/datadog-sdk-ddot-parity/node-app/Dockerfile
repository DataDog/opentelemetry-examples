FROM node:22-slim

WORKDIR /app
COPY package.json ./
RUN npm install

COPY app.js ./
COPY probe-target.js ./
COPY probes.json ./

EXPOSE 5000
CMD ["node", "app.js"]
