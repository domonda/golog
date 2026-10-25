#!/bin/bash

# Tags the root, goslog/, and logsentry/ modules with the version read from the
# VERSION file (instead of taking it as a command-line argument).

set -e

# Just in case the script is run from another directory
SCRIPT_DIR=$(cd -P -- "$(dirname -- "$0")" && pwd -P)
cd "$SCRIPT_DIR"

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

MESSAGE="${1:-$VERSION}" # optional tag message provided as the first argument

TAGS=()
for PREFIX in "${MODULE_PATHS[@]}"; do
    TAGS+=("${PREFIX}${VERSION}")
done

# Refuse to retag a released version: an existing tag would make
# "git tag" fail, and reverting below would delete the existing release tag
EXISTING=()
for TAG in "${TAGS[@]}"; do
    if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
        EXISTING+=("$TAG (local)")
    fi
    if git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null; then
        EXISTING+=("$TAG (origin)")
    elif [ $? -ne 2 ]; then
        echo "Error: could not list the tags on origin"
        exit 1
    fi
done
if [ ${#EXISTING[@]} -gt 0 ]; then
    echo "Error: $VERSION is already tagged:"
    printf '  %s\n' "${EXISTING[@]}"
    echo "Update the VERSION file before tagging a new release."
    exit 1
fi

# Only tags created by this run are ever deleted
CREATED=()
delete_created_tags() {
    for TAG in "${CREATED[@]}"; do
        git tag -d "$TAG"
    done
}

echo "Tagging $VERSION (from VERSION file) with message: '$MESSAGE'"

for TAG in "${TAGS[@]}"; do
    echo "  tag $TAG"
    if ! git tag -a "$TAG" -m "$MESSAGE"; then
        delete_created_tags
        echo "Error: could not create tag $TAG, reverted the tags created so far"
        exit 1
    fi
    CREATED+=("$TAG")
done

# Push only the new tags, never every local tag
REFS=()
for TAG in "${TAGS[@]}"; do
    REFS+=("refs/tags/$TAG")
done

echo "Tags to be pushed"
if ! git push --dry-run --atomic origin "${REFS[@]}"; then
    delete_created_tags
    echo "Error: push dry run failed, reverted local $VERSION tags"
    exit 1
fi

echo "Do you want to push tags to origin? (y/n)"
read -r CONFIRM || CONFIRM=""
if [[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]]; then
    if ! git push --atomic origin "${REFS[@]}"; then
        delete_created_tags
        echo "Error: push failed, reverted local $VERSION tags"
        exit 1
    fi
else
    delete_created_tags
    echo "Reverted local $VERSION tags"
fi
