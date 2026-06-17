#!/usr/bin/env bash

set -euo pipefail

script_dir="$(dirname "${BASH_SOURCE[0]}")"

# shellcheck source=/dev/null
source "$script_dir/.versions.env"

repo_root="$(cd "$script_dir/.." && pwd)"
chart_dir="${CHART_DIR:-$repo_root/charts/fleetconfig-controller}"
crds_dir="${chart_dir%/}/crds"
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

if [[ -z "${chart_dir:-}" || "$crds_dir" != */fleetconfig-controller/crds ]]; then
    echo "error: refusing to refresh CRDs at unsafe path: ${crds_dir:-<unset>}" >&2
    exit 1
fi

rm -rf "$crds_dir"
mkdir -p "$crds_dir"

cp "$tmp_dir/$ocm_asset_dir"/cluster/v1beta1/*.crd.yaml "$crds_dir"
cp "$tmp_dir/$ocm_asset_dir"/cluster/v1beta2/*.crd.yaml "$crds_dir"
cp "$tmp_dir/$ocm_asset_dir"/cluster/v1/*.crd.yaml "$crds_dir"
cp "$tmp_dir/$ocm_asset_dir"/addon/v1alpha1/*.crd.yaml "$crds_dir"
cp "$tmp_dir/$ocm_asset_dir"/addon/v1beta1/*.crd.yaml "$crds_dir"
