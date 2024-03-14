#!/bin/sh

# Remove any files that could contain search results from a search API
grep -c "URL:" "${TMPDIR}"/gptscript-state* | grep -v ":0" | cut -d ':' -f1 | xargs rm

find $TMPDIR -maxdepth 1 -type f -name 'gptscript-state*' -mtime -7 | xargs -I{} aws s3 cp {} s3://gptscript-state/
