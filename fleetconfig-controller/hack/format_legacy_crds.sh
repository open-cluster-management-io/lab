#!/usr/bin/env bash

set -euo pipefail

chart_dir=./charts/fleetconfig-controller/
config_dir=./config/

# wrap in a conditional block to prevent installing the CRD when legacy controllers are disabled
# and move the CRD to the templates directory, because conditional blocks are not supported in crds/ directory
echo '{{- if .Values.enableLegacyControllers }}' > "$chart_dir/templates/crd-fleetconfig.open-cluster-management.io_fleetconfigs.yaml"
cat "$chart_dir/crds/fleetconfig.open-cluster-management.io_fleetconfigs.yaml" >> "$chart_dir/templates/crd-fleetconfig.open-cluster-management.io_fleetconfigs.yaml"
echo '{{- end }}' >> "$chart_dir/templates/crd-fleetconfig.open-cluster-management.io_fleetconfigs.yaml"

# move the CRD to the config directory. This is used for testing, since we cant install a template into EnvTest.
rm -rf "$config_dir/crds/"
mkdir -p "$config_dir/crds/"
mv "$chart_dir/crds/fleetconfig.open-cluster-management.io_fleetconfigs.yaml" "$config_dir/crds/fleetconfig.open-cluster-management.io_fleetconfigs.yaml"