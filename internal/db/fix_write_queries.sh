#!/usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
TARGET=$SCRIPT_DIR/queries.write.sql.go

sed -i 's/func (q \*Queries)/func (q \*WriteQueries)/g' $TARGET

cat >> $TARGET << 'EOF'

type WriteQueries struct {
  *Queries
}

func NewWriteQueries(db DBTX) *WriteQueries {
  return &WriteQueries{
    Queries: New(db),
  }
}
EOF
