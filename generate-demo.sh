#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
NC='\033[0m' # No Color
YELLOW='\033[0;33m'
GREEN='\033[0;32m'

# Ensure VHS is installed
if ! command -v vhs &> /dev/null; then
    echo -e "${RED}Error: VHS is not installed. Please install it first:${NC}"
    echo "  brew install vhs"
    exit 1
fi

# Check for Nerd Fonts on macOS
echo "Checking for Nerd Fonts..."
HAS_NERD_FONT=false
if [ -d "$HOME/Library/Fonts" ] || [ -d "/Library/Fonts" ]; then
    # Search for files with "Nerd" or "NF" in their name and check if stdout is non-empty
    if [ -n "$(find "$HOME/Library/Fonts" "/Library/Fonts" \( -iname "*nerd*" -o -iname "*nf*" \) -maxdepth 2 2>/dev/null)" ]; then
        HAS_NERD_FONT=true
    fi
fi

if [ "$HAS_NERD_FONT" = false ]; then
    echo -e "${YELLOW}Warning: No Nerd Fonts detected in your macOS font folders.${NC}"
    echo -e "VHS (headless Chrome) requires a Nerd Font installed on your system to render brmk's icons (e.g. folder icons, GitHub, etc.) correctly."
    echo -e "Please install JetBrains Mono Nerd Font using Homebrew, then try again:"
    echo -e "  ${GREEN}brew install --cask font-jetbrains-mono-nerd-font${NC}"
    echo ""
    read -p "Would you like me to install 'font-jetbrains-mono-nerd-font' via Homebrew now? (y/N) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Installing JetBrains Mono Nerd Font..."
        brew install --cask font-jetbrains-mono-nerd-font
    else
        echo "Aborting demo generation since icons will not render correctly without a Nerd Font."
        exit 1
    fi
fi

echo "Building brmk binary..."
go build -trimpath -ldflags "-s -w" -o brmk ./cmd/brmk

echo "Preparing demo bookmark database..."
# Replace any existing local demo bookmarks with our clean sample
./brmk import examples/demo-bookmarks.md --data demo_bookmarks.json --replace

echo "Recording terminal session to demo.gif..."
export BRMK_DATA=demo_bookmarks.json
vhs demo.tape

echo "Cleaning up temporary files..."
rm -f demo_bookmarks.json

echo -e "${GREEN}Success! demo.gif has been created.${NC}"
