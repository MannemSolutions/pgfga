#!/bin/bash
VERSION=$(echo $GITHUB_REF | cut -d / -f 3)
if [ -z "${VERSION}" ]; then
  VERSION=$(git tag | sort -V | grep '^v' | tail -n1)-devel
fi
echo -ne "package internal\n\nconst appVersion = \"$VERSION\"" > internal/version.go
