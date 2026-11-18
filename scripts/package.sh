#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
output=${1:-"$root/dist"}
mkdir -p "$output"
output=$(CDPATH= cd -- "$output" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT HUP INT TERM
version=${COLLECTOR_BUILD_VERSION:-2-dev}
mkdir -p "$stage/linux/bin" "$stage/windows/bin"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$stage/linux/collector" .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$stage/windows/collector.exe" .
cp -R bin/linux/. "$stage/linux/bin/"
cp -R bin/windows/. "$stage/windows/bin/"
chmod 755 "$stage/linux/collector" "$stage/linux/bin/"*
tar -czf "$output/linux-amd64.tar.gz" -C "$stage/linux" .
python3 - "$stage/windows" "$output/windows-amd64.zip" <<'PY'
from pathlib import Path
import sys, zipfile
root=Path(sys.argv[1])
with zipfile.ZipFile(sys.argv[2], 'w', zipfile.ZIP_DEFLATED) as archive:
    for path in sorted(root.rglob('*')):
        if path.is_file(): archive.write(path, path.relative_to(root).as_posix())
PY
(cd "$output" && sha256sum linux-amd64.tar.gz windows-amd64.zip > SHA256SUMS)
