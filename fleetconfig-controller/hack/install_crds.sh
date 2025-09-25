#!/usr/bin/env bash

set -euo pipefail

script_dir="$(dirname "${BASH_SOURCE[0]}")"
source "$script_dir/.versions.env"

chart_dir=./charts/fleetconfig-controller/
tmp_dir="$(mktemp -d)"
ocm_asset_dir="api-$OCM_VERSION"
ocm_tarball="$ocm_asset_dir.tar.tgz"

cleanup() {
    rm -rf "$tmp_dir"
}
trap cleanup EXIT

# cert-manager
curl -fsL "https://github.com/cert-manager/cert-manager/releases/download/$CERT_MANAGER_VERSION/cert-manager.crds.yaml" \
  -o "$chart_dir/crds/cert-manager-$CERT_MANAGER_VERSION-crds.yaml"

# ocm
wget "https://github.com/open-cluster-management-io/api/archive/refs/tags/v$OCM_VERSION.tar.gz" -O "$tmp_dir/$ocm_tarball"
tar -xzf "$tmp_dir/$ocm_tarball" -C "$tmp_dir"

cp "$tmp_dir/$ocm_asset_dir"/cluster/v1beta1/*.crd.yaml "$chart_dir/crds"
cp "$tmp_dir/$ocm_asset_dir"/cluster/v1beta2/*.crd.yaml "$chart_dir/crds"
cp "$tmp_dir/$ocm_asset_dir"/cluster/v1/*.crd.yaml "$chart_dir/crds"
cp "$tmp_dir/$ocm_asset_dir"/addon/v1alpha1/*.crd.yaml "$chart_dir/crds"
