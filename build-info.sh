#!/bin/bash

[ -n "$(git status --porcelain --untracked-files=no)" ] && DIRTY="-dirty"
SHORT_COMMIT="$(git rev-parse --short HEAD)"
TAG=$(git tag -l --contains HEAD | head -n 1)

if [ -n "${TAG}" ]; then
    VERSION="${TAG}${DIRTY}"
else
    VERSION="${SHORT_COMMIT}${DIRTY}"
fi

BUILD_TIME=$(date -u '+%Y-%m-%d %I:%M:%S %Z')
