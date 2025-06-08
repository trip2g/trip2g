#!/bin/bash
set -e

# Load environment variables
source /etc/trip2g.env

# Generate Litestream configuration
cat > /etc/litestream.yml <<EOF
access-key-id: ${MINIO_ACCESS_KEY_ID}
secret-access-key: ${MINIO_SECRET_KEY}
endpoint: ${MINIO_ENDPOINT}

dbs:
  - path: ${DB_FILE}
    replicas:
      - type: s3
        bucket: ${MINIO_BUCKET}
        path: trip2g.db
        region: ${MINIO_REGION:-us-east-1}
        endpoint: ${MINIO_ENDPOINT}
        sync-interval: 1s
EOF

echo "Litestream configuration generated successfully"
