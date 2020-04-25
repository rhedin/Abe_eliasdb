#!/bin/sh

# Simple script to produce a self-extracting packed binary

export packed_binary=eliasdb_packed

echo "cat \$0 | sed '1,/#### Binary ####/d' | gzip -d > ./__e" > $packed_binary
echo "chmod ugo+x ./__e" >> $packed_binary
echo "mv ./__e ./\$0" >> $packed_binary
echo "./\$0" >> $packed_binary
echo "exit 0" >> $packed_binary
echo "This is a simple shell script trying to unpack the binary data" >> $packed_binary
echo "after the marker below. Unpack manually by deleting all lines" >> $packed_binary
echo "up to and including the marker line and do a gzip -d on the" >> $packed_binary
echo "binary data" >> $packed_binary
echo "#### Binary ####" >> $packed_binary
gzip -c eliasdb >> $packed_binary
chmod ugo+x $packed_binary
