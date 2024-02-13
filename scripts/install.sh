#!/bin/sh
set -e
set -o noglob

# Usage:
#   curl ... | ENV_VAR=... sh -
#     or
#   ENV_VAR=... ./install.sh
#
# Example:
#   Installing the most recent release:
#   curl ... | sh -
#
# Environment variables:
#   - INSTALL_GPTSCRIPT_SKIP_DOWNLOAD
#   If set to true will not download gptscript hash or binary.
#
#   - INSTALL_GPTSCRIPT_SYMLINK
#   If set to 'skip' will not create symlinks, 'force' will overwrite,
#   default will symlink if command does not exist in path.
#
#   - INSTALL_GPTSCRIPT_BIN_DIR
#   Directory to install gptscript binary, links, and uninstall script to, or use
#   /usr/local/bin as the default
#
#   - INSTALL_GPTSCRIPT_BIN_DIR_READ_ONLY
#   If set to true will not write files to INSTALL_GPTSCRIPT_BIN_DIR, forces
#   setting INSTALL_GPTSCRIPT_SKIP_DOWNLOAD=true

GITHUB_URL=https://github.com/gptscript-ai/gptscript/releases
DOWNLOADER=
SHA=
ARCH=
SUFFIX=
EXT=
SUDO=sudo

# --- helper functions for logs ---
info() {
  echo '[INFO] ' "$@"
}

warn() {
  echo '[WARN] ' "$@" >&2
}

fatal() {
  echo '[ERROR] ' "$@" >&2
  exit 1
}

# --- define needed environment variables ---
setup_env() {
  # --- don't use sudo if we are already root ---
  if [ $(id -u) -eq 0 ]; then
    SUDO=
  fi

  # --- use binary install directory if defined or create default ---
  if [ -n "${INSTALL_GPTSCRIPT_BIN_DIR}" ]; then
    BIN_DIR=${INSTALL_GPTSCRIPT_BIN_DIR}
  else
    # --- use /usr/local/bin if root can write to it, otherwise use /opt/bin if it exists
    BIN_DIR=/usr/local/bin
    if ! $SUDO sh -c "touch ${BIN_DIR}/gptscript-ro-test && rm -rf ${BIN_DIR}/gptscript-ro-test"; then
      if [ -d /opt/bin ]; then
        BIN_DIR=/opt/bin
      fi
    fi
  fi

  # --- if bin directory is read only skip download ---
  if [ "${INSTALL_GPTSCRIPT_BIN_DIR_READ_ONLY}" = true ]; then
    INSTALL_GPTSCRIPT_SKIP_DOWNLOAD=true
  fi
}

# --- check if skip download environment variable set ---
can_skip_download() {
  if [ "${INSTALL_GPTSCRIPT_SKIP_DOWNLOAD}" != true ]; then
    return 1
  fi
}

# --- verify an executable gptscript binary is installed ---
verify_gptscript_is_executable() {
  if [ ! -x ${BIN_DIR}/gptscript ]; then
    fatal "Executable gptscript binary not found at ${BIN_DIR}/gptscript"
  fi
}

# --- set arch and suffix, fatal if architecture not supported ---
setup_verify_arch() {
  if [ -z "$ARCH" ]; then
    PLATFORM=$(uname)
    EXT=".tar.gz"

    case $PLATFORM in
      Linux)
        PLATFORM="linux"
        ;;
      Darwin)
        PLATFORM="macOS"
        ARCH=universal
        ;;
      Windows)
        PLATFORM="windows"
        EXT=".zip"
        ;;
      *)
        fatal "Unsupported platform $PLATFORM"
    esac
  fi

  if [ -z "$ARCH" ]; then
    ARCH=$(uname -m)

    case $ARCH in
      amd64)
        ARCH=amd64
        ;;
      x86_64)
        ARCH=amd64
        ;;
      arm64)
        ARCH=arm64
        ;;
      aarch64)
        ARCH=arm64
        ;;
      *)
        fatal "Unsupported architecture $ARCH"
    esac
  fi

  SUFFIX=-${PLATFORM}-${ARCH}
}

# --- verify existence of network downloader executable ---
verify_downloader() {
  # Return failure if it doesn't exist or is no executable
  [ -x "$(command -v $1)" ] || return 1

  # Set verified executable as our downloader program and return success
  DOWNLOADER=$1
  return 0
}

verify_sha() {
  # Return failure if it doesn't exist or is no executable
  [ -x "$(command -v $1)" ] || return 1

  # Set verified executable as our sha program and return success
  SHA=$1
  return 0
}

get_sha() {
  if [ "${SHA}" = "shasum" ]; then
    $SHA -a 256 $1
  else
    $SHA $1
  fi
}

# --- create temporary directory and cleanup when done ---
setup_tmp() {
  TMP_DIR=$(mktemp -d -t gptscript-install.XXXXXXXXXX)
  TMP_HASH=${TMP_DIR}/gptscript.hash
  TMP_ARCHIVE=${TMP_DIR}/gptscript${EXT}
  cleanup() {
    code=$?
    set +e
    trap - EXIT
    rm -rf ${TMP_DIR}
    exit $code
  }
  trap cleanup INT EXIT
}

# --- use desired latest gptscript version if defined or find version from channel ---
get_release_version() {
  info "Finding latest release"
  version_url="${GITHUB_URL}/latest"
  case $DOWNLOADER in
    curl)
      VERSION_GPTSCRIPT=$(curl -w '%{url_effective}' -L -s -S ${version_url} -o /dev/null | sed -e 's|.*/||')
      ;;
    wget)
      VERSION_GPTSCRIPT=$(wget -SqO /dev/null ${version_url} 2>&1 | grep -i Location | sed -e 's|.*/||')
      ;;
    *)
      fatal "Incorrect downloader executable '$DOWNLOADER'"
      ;;
  esac
  info "Using ${VERSION_GPTSCRIPT} as release"
}

# --- download from github url ---
download() {
  [ $# -eq 2 ] || fatal 'download needs exactly 2 arguments'

  case $DOWNLOADER in
    curl)
      curl -o $1 -sfL $2
      ;;
    wget)
      wget -qO $1 $2
      ;;
    *)
      fatal "Incorrect executable '$DOWNLOADER'"
      ;;
  esac

  # Abort if download command failed
  [ $? -eq 0 ] || fatal 'Download failed'
}

# --- download hash from github url ---
download_hash() {
  HASH_URL=${GITHUB_URL}/download/${VERSION_GPTSCRIPT}/checksums.txt
  info "Downloading hash ${HASH_URL}"
  download ${TMP_HASH} ${HASH_URL}
  HASH_EXPECTED=$(grep " gptscript-${VERSION_GPTSCRIPT}${SUFFIX}${EXT}" ${TMP_HASH})
  HASH_EXPECTED=${HASH_EXPECTED%%[[:blank:]]*}
}

# --- check hash against installed version ---
installed_hash_matches() {
  if [ -x ${BIN_DIR}/gptscript ]; then
    HASH_INSTALLED=$(get_sha "${BIN_DIR}/gptscript")
    HASH_INSTALLED=${HASH_INSTALLED%%[[:blank:]]*}
    if [ "${HASH_EXPECTED}" = "${HASH_INSTALLED}" ]; then
      return
    fi
  fi
  return 1
}

# --- download archive from github url ---
download_archive() {
  ARCHIVE_URL=${GITHUB_URL}/download/${VERSION_GPTSCRIPT}/gptscript-${VERSION_GPTSCRIPT}${SUFFIX}${EXT}
  info "Downloading archive ${ARCHIVE_URL}"
  download ${TMP_ARCHIVE} ${ARCHIVE_URL}
}

# --- verify downloaded archive hash ---
verify_archive() {
  info "Verifying binary download"
  HASH_BIN=$(get_sha $TMP_ARCHIVE)
  HASH_BIN=${HASH_BIN%%[[:blank:]]*}
  if [ "${HASH_EXPECTED}" != "${HASH_BIN}" ]; then
    fatal "Download sha256 does not match ${HASH_EXPECTED}, got ${HASH_BIN}"
  fi
}

expand_archive() {
  if [ "${EXT}" = ".zip" ]; then
    unzip ${TMP_ARCHIVE} -d ${TMP_DIR}
  else
    tar xzf ${TMP_ARCHIVE} -C ${TMP_DIR}
  fi

  TMP_BIN=${TMP_DIR}/gptscript
}

# --- setup permissions and move binary to system directory ---
setup_binary() {
  chmod 755 ${TMP_BIN}
  info "Installing gptscript to ${BIN_DIR}/gptscript"
  $SUDO chown root ${TMP_BIN}
  $SUDO mv -f ${TMP_BIN} ${BIN_DIR}/gptscript
}

# --- download and verify gptscript ---
download_and_verify() {
  if can_skip_download; then
     info 'Skipping gptscript download and verify'
     verify_gptscript_is_executable
     return
  fi

  setup_verify_arch
  verify_downloader curl || verify_downloader wget || fatal 'Can not find curl or wget for downloading files'
  verify_sha sha256sum || verify_sha shasum || fatal 'Can not find sha256sum or shasum for verifying files'
  setup_tmp
  get_release_version
  download_hash

  if installed_hash_matches; then
    info 'Skipping binary download, installed gptscript matches hash'
    return
  fi

  download_archive
  verify_archive
  expand_archive
  setup_binary
}

# --- run the install process --
{
  setup_env "$@"
  download_and_verify
}
