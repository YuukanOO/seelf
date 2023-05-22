FROM node:18-alpine
WORKDIR /app
COPY package*.json .
RUN npm ci
COPY . .
RUN npm run build
WORKDIR /app/build
EXPOSE 3000
CMD ["node", "index.js"]