#!/usr/bin/env bash
set -eo pipefail
if [[ "$tag" == *"-pre-"* ]]
then
    prerelease="--prerelease"
else
    prerelease=""
fi
gh release create "$tag" \
    $prerelease \
    --repo="$GITHUB_REPOSITORY" \
    --title="${tag}" \
    --generate-notes
