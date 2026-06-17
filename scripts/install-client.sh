#!/usr/bin/env bash
# asyou Client Install Script (Linux/macOS)
# Usage: bash <(curl -sL https://raw.githubusercontent.com/Karanzhang/asyou/main/scripts/install-client.sh)
# Or: ASYOU_SERVER=https://your-server.com bash <(curl -sL ...)

set -e

ASYOU_SERVER="${ASYOU_SERVER:-https://asyou.karanz.com}"
BINDIR="${BINDIR:-/usr/local/bin}"

echo "=== asyou Client Installer ==="
echo "Server: $ASYOU_SERVER"
echo ""

# Step 1: Check prerequisites
echo "[1/5] Checking prerequisites..."
if ! command -v go &>/dev/null; then
    echo "  ❌ Go not found. Installing..."
    if command -v apt &>/dev/null; then
        sudo apt update && sudo apt install -y golang-go
    elif command -v brew &>/dev/null; then
        brew install go
    else
        echo "  Please install Go manually: https://go.dev/dl/"
        exit 1
    fi
fi
echo "  ✅ Go: $(go version)"

if ! command -v git &>/dev/null; then
    echo "  Installing git..."
    if command -v apt &>/dev/null; then
        sudo apt install -y git
    elif command -v brew &>/dev/null; then
        brew install git
    fi
fi
echo "  ✅ Git: $(git --version)"

# Step 2: Get frpc version
echo "[2/5] Getting frpc version..."
FRPC_VER=$(curl -s "$ASYOU_SERVER/api/v1/version" | python3 -c "import sys,json; print(json.load(sys.stdin)['recommended_frpc_version'])" 2>/dev/null || echo "0.69.1")
echo "  ✅ frpc version: $FRPC_VER"

# Step 3: Install frpc
echo "[3/5] Installing frpc v$FRPC_VER..."
ARCH="linux_amd64"
if [[ "$(uname)" == "Darwin" ]]; then
    ARCH="darwin_amd64"
fi

cd /tmp
curl -sL "https://github.com/fatedier/frp/releases/download/v${FRPC_VER}/frp_${FRPC_VER}_${ARCH}.tar.gz" -o frp.tar.gz
tar xzf frp.tar.gz
sudo cp "frp_${FRPC_VER}_${ARCH}/frpc" "$BINDIR/frpc"
sudo chmod +x "$BINDIR/frpc"
rm -rf "frp_${FRPC_VER}_${ARCH}" frp.tar.gz
echo "  ✅ frpc installed: $BINDIR/frpc ($(frpc --version))"

# Step 4: Build asyou CLI
echo "[4/5] Building asyou CLI..."
TMPDIR=$(mktemp -d)
git clone --depth 1 https://github.com/Karanzhang/asyou.git "$TMPDIR" 2>/dev/null || {
    echo "  ⚠️  Cannot clone from GitHub. Trying local source..."
    if [ -f cli/main.go ]; then
        go build -o "$BINDIR/asyou" ./cli
    else
        echo "  ❌ Cannot build CLI"
        exit 1
    fi
}
cd "$TMPDIR/cli" && go build -o "$BINDIR/asyou" .
sudo chmod +x "$BINDIR/asyou"
rm -rf "$TMPDIR"
echo "  ✅ CLI installed: $BINDIR/asyou"

# Step 5: Verify
echo "[5/5] Verifying..."
asyou --help >/dev/null 2>&1 && echo "  ✅ asyou CLI ready!"
frpc --version >/dev/null 2>&1 && echo "  ✅ frpc ready!"

echo ""
echo "=== Installation Complete! ==="
echo ""
echo "Next steps:"
echo "  1. Login:     asyou login --s $ASYOU_SERVER <email> <password>"
echo "  2. Expose:    asyou expose 3000 --n my-app"
echo "  3. Check:     asyou list"
echo ""
