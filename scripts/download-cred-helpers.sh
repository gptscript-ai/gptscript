#!/usr/bin/env bash

set -euo pipefail

# This script downloads the gptscript-credential-helpers. (For use in CI.)

GPTSCRIPT_CRED_HELPERS_VERSION="v0.1.0"
BINARY_DIR="binaries"

mkdir -p "$BINARY_DIR"
cd "$BINARY_DIR"

wget -O gptscript-credential-osxkeychain "https://github.com/gptscript-ai/gptscript-credential-helpers/releases/download/${GPTSCRIPT_CRED_HELPERS_VERSION}/gptscript-credential-osxkeychain"
chmod +x gptscript-credential-osxkeychain

wget -O gptscript-credential-wincred-amd64.exe "https://github.com/gptscript-ai/gptscript-credential-helpers/releases/download/${GPTSCRIPT_CRED_HELPERS_VERSION}/gptscript-credential-wincred-${GPTSCRIPT_CRED_HELPERS_VERSION}.windows-amd64.exe"
chmod +x gptscript-credential-wincred-amd64.exe

wget -O gptscript-credential-wincred-arm64.exe "https://github.com/gptscript-ai/gptscript-credential-helpers/releases/download/${GPTSCRIPT_CRED_HELPERS_VERSION}/gptscript-credential-wincred-${GPTSCRIPT_CRED_HELPERS_VERSION}.windows-arm64.exe"
chmod +x gptscript-credential-wincred-arm64.exe
