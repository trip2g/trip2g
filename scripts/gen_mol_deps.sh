#!/bin/sh

set -e

DIR=assets/ui/externaldeps
FILE=$DIR/list.view.tree

mkdir -p $DIR

echo '$trip2g_externaldeps $mol_view' > $FILE
echo '\tprop 0' >> $FILE

# collect all mol components
grep -Roh --exclude-dir='-' --exclude-dir='-view.tree' --exclude=externaldeps '\$mol_[a-z_]*' assets/ui | \
  grep -v '__' | \
  sort -u | \
  awk '{name=substr($0,2); print "\t- " $0}' >> $FILE

# mam requires a trailing newline
echo '' >> $FILE
