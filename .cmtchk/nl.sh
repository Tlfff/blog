#!/bin/zsh
for f in "$@"; do echo "===== $f ====="; nl -ba -w4 -s'| ' "$f"; done
