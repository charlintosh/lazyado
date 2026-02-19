#!/bin/bash

# Sync .agents/skills/ to all supported folders via symlinks
# Source: .agents/skills/
# Targets: folders listed in .agents.support.json

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$PROJECT_ROOT/.agents.support.json"
SOURCE_DIR="$PROJECT_ROOT/.agents/skills"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Syncing .agents/skills/ to supported folders..."
echo ""

# Check if config file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo -e "${RED}Error: $CONFIG_FILE not found${NC}"
    exit 1
fi

# Check if source directory exists
if [[ ! -d "$SOURCE_DIR" ]]; then
    echo -e "${RED}Error: $SOURCE_DIR not found${NC}"
    exit 1
fi

# Parse folders from JSON using grep/sed (no jq dependency)
FOLDERS=$(grep -o '"[^"]*"' "$CONFIG_FILE" | grep -v 'folders' | tr -d '"')

for folder in $FOLDERS; do
    TARGET_PARENT="$PROJECT_ROOT/$folder"
    TARGET_DIR="$TARGET_PARENT/skills"

    echo -n "  $folder/skills/ "

    # Create parent folder if it doesn't exist
    if [[ ! -d "$TARGET_PARENT" ]]; then
        mkdir -p "$TARGET_PARENT"
        echo -n "(created parent) "
    fi

    # Remove existing directory or symlink
    if [[ -L "$TARGET_DIR" ]]; then
        rm "$TARGET_DIR"
        echo -n "(removed old symlink) "
    elif [[ -d "$TARGET_DIR" ]]; then
        rm -rf "$TARGET_DIR"
        echo -n "(removed old directory) "
    fi

    # Create relative symlink
    RELATIVE_PATH="../.agents/skills"
    ln -s "$RELATIVE_PATH" "$TARGET_DIR"

    echo -e "${GREEN}-> symlinked${NC}"
done

echo ""
echo -e "${GREEN}Done!${NC} All skills folders are now symlinked to .agents/skills/"
