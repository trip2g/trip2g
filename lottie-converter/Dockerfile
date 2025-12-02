FROM node:20-slim

LABEL org.opencontainers.image.title="Lottie Converter" \
      org.opencontainers.image.description="Convert Telegram custom emoji (TGS/WEBM/WEBP) to GIF" \
      org.opencontainers.image.authors="alexes" \
      org.opencontainers.image.version="0.1" \
      org.opencontainers.image.source="https://github.com/alexes/lottie-converter"

RUN apt-get update && apt-get install -y \
    ffmpeg \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --only=production && npm cache clean --force

COPY server.js thumbnail.js cache.js ./

EXPOSE 3000

ENV NODE_ENV=production

CMD ["node", "server.js"]
