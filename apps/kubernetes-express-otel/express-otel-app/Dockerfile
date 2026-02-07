FROM node:20.18.1-slim

WORKDIR /home/express-otel-app

COPY package.json package-lock.json ./

RUN npm ci --omit=dev

COPY . .

EXPOSE 3000

CMD ["npm", "start"]