# Build and Publish Docker Image

## Build image for AMD64 (production servers)

```bash
cd lottie-converter
docker buildx build --platform linux/amd64 -t alexes/lottie:v0.1 -t alexes/lottie:latest --load .
```

## Build multi-platform image (AMD64 + ARM64)

```bash
cd lottie-converter
docker buildx build --platform linux/amd64,linux/arm64 -t alexes/lottie:v0.1 -t alexes/lottie:latest --push .
```

Note: `--push` is required for multi-platform builds

## Login to Docker Hub

```bash
docker login
```

## Push to Docker Hub

```bash
docker push alexes/lottie:v0.1
docker push alexes/lottie:latest
```

## Run locally for testing

```bash
docker run -d \
  --name lottie-converter \
  -p 3000:3000 \
  -v ./cache:/app/cache \
  --env-file .env \
  alexes/lottie:v0.1
```

Where `.env` contains:
```
TELEGRAM_BOT_TOKEN=your_bot_token_here
SERVER_URL=https://your-domain.com
NODE_ENV=production
```
