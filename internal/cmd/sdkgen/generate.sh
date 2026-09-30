#!/bin/sh
set -eu
sdk_root=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)
cd "$sdk_root"
upstream=github.com/gophercloud/gophercloud/v2
version=$(go list -m -f '{{.Version}}' "$upstream")
if [ "$version" != v2.15.0 ]; then
    echo "sdkgen supports v2.15.0; update the generator and contract tests before changing the dependency" >&2
    exit 1
fi
go mod download "$upstream"
upstream_dir=$(go list -m -f '{{.Dir}}' "$upstream")
generation_tmp=$(mktemp -d)
trap 'rm -rf "$generation_tmp"' EXIT HUP INT TERM
(cd "$upstream_dir" && go list -mod=readonly -deps -export -json ./openstack/... > "$generation_tmp/packages.json")
go run ./internal/cmd/sdkgen -metadata "$generation_tmp/packages.json" -output "$sdk_root"
