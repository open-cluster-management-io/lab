#!/bin/bash

set -euo pipefail

# Default values
branch="main"
port="8000"
local_bundle=""

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Download and serve a support bundle from GitHub Actions artifacts, or serve a local bundle.

OPTIONS:
    -b, --branch BRANCH     Git branch to get support bundle from (default: main)
    -p, --port PORT         Port to serve HTTP server on (default: 8000)
    -l, --local [PATH]      Serve an existing local bundle directory (default: ./fleetconfig-support-bundle)
    -h, --help              Show this help message

EXAMPLES:
    $0                                    # Download from main branch, serve on port 8000
    $0 -b feature-branch                  # Download from feature-branch
    $0 -b main -p 8080                    # Download from main branch, serve on port 8080
    $0 -l                                 # Serve existing ./fleetconfig-support-bundle directory
    $0 -l /path/to/bundle                 # Serve specific local bundle directory. Must contain 'hub' and 'spoke' subdirectories.
    $0 -l /path/to/bundle -p 9000         # Serve local bundle on port 9000

NOTES:
    When using --local, the directory should contain 'hub' and 'spoke' subdirectories.
    Cannot use subdirectories within fleetconfig-support-bundle as the local path.

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -b|--branch)
            branch="$2"
            shift 2
            ;;
        -p|--port)
            port="$2"
            shift 2
            ;;
        -l|--local)
            # Check if next argument exists and doesn't start with -
            if [[ $# -gt 1 && ! "$2" =~ ^- ]]; then
                local_bundle="$2"
                shift 2
            else
                local_bundle="./fleetconfig-support-bundle"
                shift 1
            fi
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Error: Unknown option $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

# Validate arguments
if [[ ! "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 ]] || [[ "$port" -gt 65535 ]]; then
    echo "Error: Port must be a number between 1 and 65535" >&2
    exit 1
fi

# Validate local bundle or branch
if [[ -n "$local_bundle" ]]; then
    if [[ -n "$branch" && "$branch" != "main" ]]; then
        echo "Warning: --branch option ignored when using --local" >&2
    fi

    if [[ ! -d "$local_bundle" ]]; then
        echo "Error: Local bundle directory '$local_bundle' does not exist" >&2
        exit 1
    fi

    # Convert to absolute path for comparison
    local_bundle_abs=$(realpath "$local_bundle")
    current_dir=$(pwd)
    fleetconfig_bundle_dir="$current_dir/fleetconfig-support-bundle"

    # Check if trying to use a subdirectory of fleetconfig-support-bundle
    if [[ "$local_bundle_abs" == "$fleetconfig_bundle_dir"/* ]]; then
        echo "Error: Cannot use subdirectories within fleetconfig-support-bundle as the local path" >&2
        echo "Local path: $local_bundle_abs" >&2
        echo "Use the full fleetconfig-support-bundle directory instead, or a path outside of it" >&2
        exit 1
    fi

    if [[ ! -d "$local_bundle/hub" || ! -d "$local_bundle/spoke" ]]; then
        echo "Error: Local bundle directory must contain 'hub' and 'spoke' subdirectories" >&2
        echo "Found directories in '$local_bundle':" >&2
        ls -la "$local_bundle" >&2
        exit 1
    fi
else
    if [[ -z "$branch" ]]; then
        echo "Error: Branch cannot be empty" >&2
        exit 1
    fi
fi

cleanup() {
    pkill -f "python3 -m http.server" 2>/dev/null || true
    echo "Killed HTTP server on port $port"
    exit 0
}

# Set trap to call cleanup on script termination
trap cleanup SIGINT SIGTERM EXIT

# Function to copy index.html template
copy_index_html() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cp "$script_dir/index.html" fleetconfig-support-bundle/index.html
    cp "$script_dir/styles.css" fleetconfig-support-bundle/styles.css
    cp "$script_dir/script.js" fleetconfig-support-bundle/script.js
}

if [[ -n "$local_bundle" ]]; then
    # Local bundle mode
    echo "Using local bundle: $local_bundle"

    # Convert to absolute path
    local_bundle=$(realpath "$local_bundle")
    current_dir=$(pwd)
    fleetconfig_bundle_dir="$current_dir/fleetconfig-support-bundle"

    if [[ "$local_bundle" == "$fleetconfig_bundle_dir" ]]; then
        # Bundle is exactly the fleetconfig-support-bundle directory
        echo "Bundle is already in the expected location"
    else
        # Bundle is outside fleetconfig-support-bundle, create symlink
        if [[ -d "fleetconfig-support-bundle" ]]; then
            rm -rf fleetconfig-support-bundle
        fi
        ln -s "$local_bundle" fleetconfig-support-bundle
    fi

    copy_index_html

    echo "Starting HTTP server at http://localhost:$port"
    echo "Serving local bundle: $local_bundle"
    echo "Press Ctrl+C to stop the server and exit"
    cd fleetconfig-support-bundle && python3 -m http.server "$port"
else
    # Download mode
    bundle_url=$(gh api -X GET \
      'repos/open-cluster-management-io/lab/actions/artifacts?per_page=100&page=1' | \
      jq -r '.artifacts[] | select(.name == "bundle-files" and .workflow_run.head_branch == "'"$branch"'") | .archive_download_url' | head -n 1)

    if [ -z "$bundle_url" ]; then
      echo "No support bundle found for branch $branch"
      exit 1
    fi

    mkdir -p fleetconfig-support-bundle
    rm -rf fleetconfig-support-bundle/*
    mkdir -p fleetconfig-support-bundle/hub fleetconfig-support-bundle/spoke

    echo "Downloading support bundle from branch '$branch'..."
    gh api -X GET "$bundle_url" > fleetconfig-support-bundle/bundle.tar.gz

    tar -xzf fleetconfig-support-bundle/bundle.tar.gz -C fleetconfig-support-bundle
    tar -xzf fleetconfig-support-bundle/hub-bundle-*.tar.gz -C fleetconfig-support-bundle/hub
    tar -xzf fleetconfig-support-bundle/spoke-bundle-*.tar.gz -C fleetconfig-support-bundle/spoke
    rm -f fleetconfig-support-bundle/bundle.tar.gz fleetconfig-support-bundle/hub-bundle-*.tar.gz fleetconfig-support-bundle/spoke-bundle-*.tar.gz

    copy_index_html

    echo "Starting HTTP server at http://localhost:$port"
    echo "Press Ctrl+C to stop the server and exit"
    cd fleetconfig-support-bundle && python3 -m http.server "$port"
fi