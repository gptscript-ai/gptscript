#!/bin/sh

find $TMPDIR -maxdepth 1 -type f -name 'gptscript-state*' -mtime -7 | xargs -I{} aws s3 cp {} s3://gptscript-state/
