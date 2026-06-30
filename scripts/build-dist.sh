#!/usr/bin/env bash
set -euo pipefail

out_dir="${1:-dist}"
targets="${APULSE_TARGETS:-darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64}"

rm -rf "$out_dir"
mkdir -p "$out_dir"

for target in $targets; do
  os="${target%/*}"
  arch="${target#*/}"
  ext=""
  if [[ "$os" == "windows" ]]; then
    ext=".exe"
  fi

  output="$out_dir/apulse_${os}_${arch}${ext}"
  echo "building $output"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "$output" ./cmd/apulse
done

checksum_file="$out_dir/checksums.txt"
: > "$checksum_file"
for asset in "$out_dir"/apulse_*; do
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$out_dir" && sha256sum "$(basename "$asset")") >> "$checksum_file"
  else
    (cd "$out_dir" && shasum -a 256 "$(basename "$asset")") >> "$checksum_file"
  fi
done
