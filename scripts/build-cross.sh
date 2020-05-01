#!/usr/bin/env bash
set -euo pipefail

for plat in linux-amd64 linux-arm64 freebsd-amd64; do
  echo "${plat}"
  oa=(${plat/-/ })
  GOOS="${oa[0]}" GOARCH="${oa[1]}" go build -v -trimpath -o "build/runitor-${plat}"
done
