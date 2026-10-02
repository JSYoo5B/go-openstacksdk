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
# The generator reads this checkout as AST data; it never imports OpenStackSDK.
python_source=${SDKGEN_OPENSTACKSDK_SOURCE:-}
if [ -z "$python_source" ]; then
    python_source="$generation_tmp/openstacksdk"
    git init -q "$python_source"
    git -C "$python_source" fetch -q --depth=1 https://github.com/openstack/openstacksdk.git ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe
    git -C "$python_source" checkout -q --detach FETCH_HEAD
fi
(cd "$upstream_dir" && go list -mod=readonly -deps -export -json ./openstack/... > "$generation_tmp/packages.json")
go run ./internal/cmd/sdkgen -metadata "$generation_tmp/packages.json" -output "$sdk_root" -openstacksdk-source "$python_source"
