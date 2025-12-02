FROM node:20-slim

RUN apt-get update && apt-get install -y \
    ffmpeg \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY package.json ./
RUN npm install --only=production
COPY . .

EXPOSE 3000
CMD ["node", "server.js"]
