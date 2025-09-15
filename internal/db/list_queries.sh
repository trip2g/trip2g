#!/usr/bin/env bash

set -euo pipefail

READ_FILE="queries.read.sql"
WRITE_FILE="queries.write.sql"

status=0

echo "Checking $READ_FILE ..."
# Look for forbidden write operations inside read.sql
if grep -niE '(^|[^a-z])(insert into|update|delete)([^a-z]|$)' "$READ_FILE"; then
  echo "❌ ERROR: $READ_FILE contains write operations (INSERT/UPDATE/DELETE)."
  status=1
else
  echo "✅ $READ_FILE passed."
fi

echo "Checking $WRITE_FILE ..."
# Use awk to detect if any statement *starts* with SELECT
if ! awk '
BEGIN { in_stmt=0; stmt_type=""; bad=0 }
/^[[:space:]]*--/ { next }       # skip comments
/^[[:space:]]*$/  { next }       # skip blank lines

{
  if (in_stmt == 0) {
    # new statement starts here
    first=$1
    low=tolower(first)
    if (low == "select") {
      print "❌ ERROR: top-level SELECT at line " NR ": " $0
      bad=1
    }
    in_stmt=1
  }
  if ($0 ~ /;/) {
    # statement ended
    in_stmt=0
  }
}
END { exit bad }
' "$WRITE_FILE"
then
  status=1
else
  echo "✅ $WRITE_FILE passed."
fi

exit $status

