#!/bin/bash

# Tags the root, goslog/, and logsentry/ modules with the version read from the
# VERSION file (instead of taking it as a command-line argument).

# Just in case the script is run from another directory
SCRIPT_DIR=$(cd -P -- $(dirname -- "$0") && pwd -P)
cd $SCRIPT_DIR

MODULE_PATHS=("" "goslog/" "logsentry/")

VERSION_FILE="$SCRIPT_DIR/VERSION"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: VERSION file not found at $VERSION_FILE"
    exit 1
fi

# Read the version from the VERSION file, trimming surrounding whitespace
VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")

if [ -z "$VERSION" ]; then
    echo "Error: VERSION file is empty"
    exit 1
fi

SEMVER_REGEX='^v([0-9]+)\.([0-9]+)\.([0-9]+)(-[a-zA-Z0-9]+)?$'
if [[ ! "$VERSION" =~ $SEMVER_REGEX ]]; then
    echo "Error: Invalid version format in VERSION file: $VERSION"
    echo ""
    echo "Version must be in format: vMAJOR.MINOR.PATCH[-PRERELEASE]"
    echo "Examples: v0.99.1, v1.0.0, v2.1.3-beta1"
    exit 1
fi

# Show current release tags for each module before tagging
echo "Current release tags:"
echo ""

for PREFIX in "${MODULE_PATHS[@]}"; do
    if [ -z "$PREFIX" ]; then
        MODULE_NAME="(root module)"
    else
        MODULE_NAME="${PREFIX%/}"
    fi
    echo "  Module: $MODULE_NAME"

    # Get the latest tag for this module
    LATEST_TAG=$(git tag -l "${PREFIX}v*" --sort=-v:refname | head -1)
    if [ -n "$LATEST_TAG" ]; then
        echo "    Latest: $LATEST_TAG"
        # Show last 3 tags for this module
        echo "    Recent:"
        git tag -l "${PREFIX}v*" --sort=-v:refname | head -3 | sed 's/^/      /'
    else
        echo "    No tags yet"
    fi
    echo ""
done

MESSAGE="$VERSION"
if [ -n "$1" ]; then
    MESSAGE="$1" # optional tag message provided as the first argument
fi

echo "Tagging $VERSION (from VERSION file) with message: '$MESSAGE'"

for PREFIX in "${MODULE_PATHS[@]}"; do
    echo "  tag ${PREFIX}${VERSION}"
    git tag -a "${PREFIX}${VERSION}" -m "$MESSAGE"
done

echo "Tags to be pushed"
git push --tags --dry-run

echo "Do you want to push tags to origin? (y/n)"
read CONFIRM
if [[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]]; then
    git push origin --tags
else
    for PREFIX in "${MODULE_PATHS[@]}"; do
        git tag -d "${PREFIX}${VERSION}"
    done
    echo "Reverted local $VERSION tags"
fi
