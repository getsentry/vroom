#!/usr/bin/env bash
set -euxo pipefail

CHANGE_DATE="$(date +'%Y-%m-%d' -d '3 years')"
echo "Bumping Change Date to $CHANGE_DATE"
sed -i -e "s/\(Change Date:\s*\)[-0-9]\+\$/\\1$CHANGE_DATE/" LICENSE
