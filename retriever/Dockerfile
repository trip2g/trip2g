FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    sentence-transformers \
    fastapi \
    uvicorn[standard]

COPY server.py .

ENV MODEL_NAME=BAAI/bge-m3
ENV RERANKER_MODEL=BAAI/bge-reranker-v2-m3
ENV HF_HOME=/data
EXPOSE 8000

CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
