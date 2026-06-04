#!/usr/bin/env bash

set -euo pipefail

script_dir="$(dirname "${BASH_SOURCE[0]}")"

# shellcheck source=/dev/null
source "$script_dir/.versions.env"

chart_dir=./charts/fleetconfig-controller/
tmp_dir="$(mktemp -d)"

_ocm_api_version="${OCM_API_VERSION#v}"
ocm_asset_dir="api-${_ocm_api_version}"
ocm_tarball="$ocm_asset_dir.tar.tgz"

cleanup() {
    rm -rf "$tmp_dir"
}
trap cleanup EXIT

# ocm
wget "https://github.com/open-cluster-management-io/api/archive/refs/tags/v${_ocm_api_version}.tar.gz" -O "$tmp_dir/$ocm_tarball"
tar -xzf "$tmp_dir/$ocm_tarball" -C "$tmp_dir"

cp "$tmp_dir/$ocm_asset_dir"/cluster/v1beta1/*.crd.yaml "$chart_dir/crds"
cp "$tmp_dir/$ocm_asset_dir"/cluster/v1beta2/*.crd.yaml "$chart_dir/crds"
cp "$tmp_dir/$ocm_asset_dir"/cluster/v1/*.crd.yaml "$chart_dir/crds"
cp "$tmp_dir/$ocm_asset_dir"/addon/v1beta1/*.crd.yaml "$chart_dir/crds"
