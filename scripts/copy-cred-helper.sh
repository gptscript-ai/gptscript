#!/usr/bin/env bash

set -euo pipefail

OS="$1"
ARCH="$2"

case "$OS" in
	"darwin")
		case "$ARCH" in
			"amd64")
				cp binaries/gptscript-credential-osxkeychain releases/default_darwin_amd64_v1
				;;
			"arm64")
				cp binaries/gptscript-credential-osxkeychain releases/default_darwin_arm64
				;;
		esac
		;;
	"windows")
		case "$ARCH" in
			"amd64")
				cp binaries/gptscript-credential-wincred.exe releases/default_windows_amd64_v1
				;;
		esac
		;;
esac
